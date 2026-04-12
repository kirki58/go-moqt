package session

import (
	"context"
	"fmt"
	"go-moq/pkg/message"
	"go-moq/pkg/model"
	"go-moq/pkg/session/control"
	"go-moq/pkg/transport"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
)

const openStreamTimeoutSecs = 5 * time.Second

// Per-session publisher, deals with the data plane
// Publisher streams tracks to their subscribers
type Publisher struct {
	sess *Session // A reference to the session this publisher belongs to

	// Active Incoming Subscription by their Track alias
	SubInAliasesMutex                 sync.Mutex
	ActiveIncomingSubscriptionAliases map[uint64]*Subscription

	// Active Incoming Subscription by their Full Track Name
	SubInNamesMutex                 sync.Mutex
	ActiveIncomingSubscriptionNames map[string]*Subscription

	latestAliasMutex sync.Mutex
	latestAlias      uint64
}

func (pub *Publisher) GetSubscriptionByName(ftn *model.MoqtFullTrackName) (*Subscription, bool) {
	pub.SubInNamesMutex.Lock()
	defer pub.SubInNamesMutex.Unlock()
	sub, ok := pub.ActiveIncomingSubscriptionNames[ftn.ToString()]
	return sub, ok
}

func (pub *Publisher) GetSubscriptionByAlias(alias uint64) (*Subscription, bool) {
	pub.SubInAliasesMutex.Lock()
	defer pub.SubInAliasesMutex.Unlock()
	sub, ok := pub.ActiveIncomingSubscriptionAliases[alias]
	return sub, ok
}

// Instantiates a subscription, returns it + the latest object location
// Will also register the subscription to maps
// returns ok if created registered sucessfully
func (pub *Publisher) NewSubscription(reqId uint64, ftn *model.MoqtFullTrackName, filter model.SubscriptionFilter,
	parameters []model.MoqtKeyValuePair, dispatcher *Dispatcher) (*Subscription, error) {
	pub.latestAliasMutex.Lock()
	trackAlias := pub.latestAlias
	pub.latestAlias++
	pub.latestAliasMutex.Unlock()

	sub := &Subscription{
		ID:            reqId,
		Alias:         trackAlias,
		FullTrackName: ftn,
		Filter:        filter,
		Status:        SubscriptionStatusEstablished,
		Parameters:    parameters,
		Publisher:     pub,
		Dispatcher:    dispatcher,
		// DispatcherChannel: dispatcherChan, // Must be assigned by the subscribe message handler
		// DropChannel:       dropChannel,    // Likewise
	}

	pub.SubInNamesMutex.Lock()
	pub.ActiveIncomingSubscriptionNames[ftn.ToString()] = sub
	pub.SubInNamesMutex.Unlock()

	pub.SubInAliasesMutex.Lock()
	pub.ActiveIncomingSubscriptionAliases[trackAlias] = sub
	pub.SubInAliasesMutex.Unlock()

	return sub, nil
}

// will return false if track is over before reading a single object (extremely unlikely)
func (pub *Publisher) LargestObjectLocation(sub *Subscription) (*model.MoqtLocation, bool) {
	largestObj, ok := <-sub.DispatcherChannel
	if !ok {
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneTrackEnded,
			StreamCount: 0,
			ErrorReason: model.NewReasonPhrase("No error - Track ended gracefully"),
		}, nil)
		return nil, false
	}
	return &largestObj.Location, true
}

// returns false when trying to read closed dispatcher channel (means track is over before joining)
func (pub *Publisher) JoinEstablishedSubscription(sub *Subscription) error {
	// Note: Only NextGroupStart join is implemented for now.
	switch sub.Filter.FilterType {
	case model.NextGroupStart:
		// Wait for next group start to join
		latestObj := <-sub.DispatcherChannel
		latestGroup := latestObj.Location.GroupId

		for {
			obj, ok := <-sub.DispatcherChannel
			if !ok {
				pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
					RequestId:   sub.ID,
					StatusCode:  control.PublishDoneTrackEnded,
					StreamCount: 0,
					ErrorReason: model.NewReasonPhrase("No error - Track ended gracefully"),
				}, nil)
				return fmt.Errorf("Track ended while waiting for next group start")
			}

			if obj.Location.GroupId > latestGroup { // reached the next group, we can join now
				go pub.publishForSubscription(sub) // start streaming to this subscription in a separate goroutine
				return nil                         // sucessfully joined
			}
		}
	default:
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneInternalError,
			StreamCount: 0,
			ErrorReason: model.NewReasonPhrase("Unsupported subscription filter type"),
		}, nil)
		return fmt.Errorf("Unsupported subscription filter type: %d\n", sub.Filter.FilterType)
	}
}

// For simplicity, it's assumed that each group will have only 1 subgroup, so a 1:1:1 mapping exists for group:subgroup:stream
func (pub *Publisher) publishForSubscription(sub *Subscription) {
	streamCount := uint64(0)  // Number of opened streams
	latestGroup := ^uint64(0) // assign it to 11111.... (64) this will indicate a newly started stream
	var latestStream transport.SendStream

	// Needed to calculate ObjectIdDelta
	// If it's a recently-joined stream or a newly opened stream (head of group) it is -1
	// Otherwise it is the Object id of the object previous to the one in the below loop
	objIdTracker := ^uint64(0) // assign it to 11111....

	// Listen to sub.DispatcherChannel until it's open (breaks out when channel is closed by the dispatcher)
Loop: // label to be used for breaking the main loop
	for { // obj := range sub.DispatcherChannel
		select {
		case obj, ok := <-sub.DispatcherChannel:
			if !ok { // channel is closed, track is over
				break Loop
			}
			if obj.Location.GroupId > latestGroup || latestGroup == ^uint64(0) { // group boundary reached, FIN the previous stream, open a new stream
				if latestGroup != ^uint64(0) { // Not recently joined so there is a previous stream
					latestStream.Close()
				}
				latestGroup = obj.Location.GroupId
				objIdTracker = ^uint64(0) // reset the id tracker
				if err := pub.startNewGroup(sub, latestStream, latestGroup, streamCount); err != nil {
					fmt.Printf("%v", err)
					return
				}
				streamCount++
			}

			// send objects over latestStream
			var objIdDelta uint64
			if objIdTracker == ^uint64(0) {
				objIdDelta = obj.Location.ObjectId // 0
			} else {
				objIdDelta = obj.Location.ObjectId - objIdTracker - 1
			}
			objIdTracker = obj.Location.ObjectId

			if err := pub.sendSubgroupObject(sub, latestStream, obj, objIdDelta, streamCount); err != nil {
				fmt.Printf("%v", err)
				return
			}

		case <-sub.DropChannel:
			// drop all objects in the current group
			// when the next group start object is received open a new stream for it, publish the first object and break
			for {
				obj, ok := <-sub.DispatcherChannel
				if !ok {
					break Loop
				}
				if obj.Location.GroupId > latestGroup {
					latestStream.Close()
					latestGroup = obj.Location.GroupId
					if err := pub.startNewGroup(sub, latestStream, latestGroup, streamCount); err != nil {
						fmt.Printf("%v", err)
						return
					}
					streamCount++

					// Since this is the first object, it's certain we are going to use it's object id as objIdDelta
					if err := pub.sendSubgroupObject(sub, latestStream, obj, obj.Location.ObjectId, streamCount); err != nil {
						fmt.Printf("%v", err)
						return
					}
					objIdTracker = obj.Location.ObjectId
					break
				}
			}
		}
	}

	// track is over send an end of track object in a new group
	latestStream.Close() // close the last group's stream
	latestGroup++        // note: ^uint64(0) + 1 = 0, so if the dispatcher channel is somehow closed before sending any objects EOT object is sent in group 0
	if err := pub.startNewGroup(sub, latestStream, latestGroup, streamCount); err != nil {
		fmt.Printf("%v", err)
		return
	}
	streamCount++

	if err := pub.sendEndOfTrackObject(sub, latestStream, streamCount); err != nil {
		fmt.Printf("%v", err)
		return
	}
}

// Before calling: increment latestGroup, close latestStream
// After calling: increment stream counter, check for errors
// internally terminates subscriptions for fatal errors
func (pub *Publisher) startNewGroup(sub *Subscription, latestStream transport.SendStream, latestGroup uint64, streamCount uint64) error {
	// open a new stream
	ctx, cancel := context.WithTimeout(pub.sess.Conn.Context(), openStreamTimeoutSecs)
	defer cancel()
	latestStream, err := pub.sess.Conn.OpenUniStreamSync(ctx)

	if err != nil {
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneInternalError,
			StreamCount: streamCount,
			ErrorReason: model.NewReasonPhrase("Failed to open data stream"),
		}, latestStream)
		return fmt.Errorf("Failed to open stream for subscription %d, error: %v\n", sub.ID, err)
	}

	// Send subgroup header over the stream
	// since this is a single-subgroup stream it's certain that end of group will be present in the stream
	// Extensions are present for every subgroup object within this stream, those who have no metadata to transmit MUST set their extensions length to 0
	sgh := model.NewSubGroupHeader(sub.Alias, latestGroup, model.SHWithEndOfGroup(), model.SHWithExtensions(), model.SHWithPublisherPriority(128))
	var sghBufArr [128]byte
	sghBuf := sghBufArr[:0] // Create a slice backed by the stack array

	message.EncodeSubgroupHeader(&sghBuf, sgh)
	n, err := latestStream.Write(sghBuf) // TODO: Might implement a timeout context wrapper for this line??

	if err != nil {
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneInternalError,
			StreamCount: streamCount,
			ErrorReason: model.NewReasonPhrase("Failed to write data to stream"),
		}, latestStream)
		return fmt.Errorf("Failed to write subgroup header for subscription %d, Written %d bytes out of %d-length message ,error: %v\n", sub.ID, n, len(sghBuf), err)
	}
	return nil
}

// Before calling this function, calculate object id delta.
// DO NOT Increment stream count after this function call
// check for errors after calling this function
// internally cleans up subscriptions in case of fatal errors
func (pub *Publisher) sendSubgroupObject(sub *Subscription, latestStream transport.SendStream, obj *model.MoqtObject, objIdDelta uint64, streamCount uint64) error {
	sgo, err := model.NewSubgroupObject(objIdDelta, model.SOWithPayload(obj.Payload), model.SOWithExtensions(obj.ExtensionHeaders), model.SOWithStatus(obj.ObjectStatus))
	if err != nil {
		// Dispatcher yielded a corrupt object, close dispatcher channel, remove subscription, close all ongoing streams, send PUBLISH_DONE
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneInternalError,
			StreamCount: streamCount,
			ErrorReason: model.NewReasonPhrase("Publisher's dispatcher handed corrupt object"),
		}, latestStream)
		return fmt.Errorf("Dispatcher handed corrupt object for subscription with id: %d, error: %v\n", sub.ID, err)
	}
	var sgoBufArr [0]byte
	sgoBuf := sgoBufArr[:0]

	message.EncodeSubgroupObject(&sgoBuf, sgo)
	n, err := latestStream.Write(sgoBuf)
	if err != nil {
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneInternalError,
			StreamCount: streamCount,
			ErrorReason: model.NewReasonPhrase("Failed to write data to stream"),
		}, latestStream)
		return fmt.Errorf("Failed to write subgroup object for subscription %d, Written %d bytes out of %d-length message ,error: %v\n", sub.ID, n, len(sgoBuf), err)
	}
	return nil
}

// DO NOT increment stream count after this function call
// check for errors after calling this function
// internally cleans up subscriptions no matter if any errors occured or not
func (pub *Publisher) sendEndOfTrackObject(sub *Subscription, latestStream transport.SendStream, streamCount uint64) error {
	eotObj, _ := model.NewSubgroupObject(0, model.SOWithStatus(model.EndOfTrack))

	var sgoBufArr [0]byte
	sgoBuf := sgoBufArr[:0]

	message.EncodeSubgroupObject(&sgoBuf, eotObj)
	n, err := latestStream.Write(sgoBuf)
	if err != nil {
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneInternalError,
			StreamCount: streamCount,
			ErrorReason: model.NewReasonPhrase("Failed to write data to stream"),
		}, latestStream)
		return fmt.Errorf("Failed to write subgroup object for subscription %d, Written %d bytes out of %d-length message ,error: %v\n", sub.ID, n, len(sgoBuf), err)
	}

	pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
		RequestId:   sub.ID,
		StatusCode:  control.PublishDoneTrackEnded,
		StreamCount: streamCount,
		ErrorReason: model.NewReasonPhrase("No error - Track ended gracefully"),
	}, latestStream)

	return nil
}

func (pub *Publisher) cleanUpSubscription(sub *Subscription, publishDone *control.PublishDoneMessage, latestStream transport.SendStream) {
	// 1. Close the Dispatcher channel and stop receiving any objects from it
	// 2. Close latestStream
	// 2. Send a PUBLISH_DONE message
	// 3. Remove the subscription from in-memory registry
	sub.Dispatcher.Close(sub)

	if latestStream != nil {
		switch publishDone.StatusCode {
		case control.PublishDoneInternalError:
			latestStream.CancelWrite(quic.StreamErrorCode(quic.InternalError)) // Aborts the stream, no retransmission
		case control.PublishDoneTrackEnded:
			latestStream.Close() // Will receive on-fly objects or objects that needs retransmitting
		}
	}

	pub.sess.Cmf.WriteControlMessage(publishDone)
	pub.removeSubscription(sub)
}

func (pub *Publisher) removeSubscription(sub *Subscription) {
	pub.SubInNamesMutex.Lock()
	delete(pub.ActiveIncomingSubscriptionNames, sub.FullTrackName.ToString())
	pub.SubInNamesMutex.Unlock()

	pub.SubInAliasesMutex.Lock()
	delete(pub.ActiveIncomingSubscriptionAliases, sub.Alias)
	pub.SubInAliasesMutex.Unlock()
}

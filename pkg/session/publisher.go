package session

import (
	"context"
	"fmt"
	moqt "go-moq"
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

	TrackRegistry moqt.TrackRegistry

	// Active Incoming Subscription by their Track alias
	SubInAliasesMutex                 sync.Mutex
	ActiveIncomingSubscriptionAliases map[uint64]*Subscription

	// Active Incoming Subscription by their Full Track Name
	SubInNamesMutex                 sync.Mutex
	ActiveIncomingSubscriptionNames map[string]*Subscription

	latestAliasMutex sync.Mutex
	latestAlias      uint64
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
	for obj := range sub.DispatcherChannel {
		if obj.Location.GroupId > latestGroup || latestGroup == ^uint64(0) { // group boundary reached, FIN the previous stream, open a new stream
			if latestGroup != ^uint64(0) { // Not recently joined so there is a previous stream
				latestStream.Close()
			}
			latestGroup = obj.Location.GroupId
			// open a new stream
			ctx, cancel := context.WithTimeout(pub.sess.Conn.Context(), openStreamTimeoutSecs)
			latestStream, err := pub.sess.Conn.OpenUniStreamSync(ctx)
			cancel()

			objIdTracker = ^uint64(0) // reset the id tracker

			if err != nil {
				fmt.Printf("Failed to open stream for subscription %d, error: %v\n", sub.ID, err)
				pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
					RequestId:   sub.ID,
					StatusCode:  control.PublishDoneInternalError,
					StreamCount: streamCount,
					ErrorReason: model.NewReasonPhrase("Failed to open data stream"),
				}, latestStream)
				return
			}

			streamCount++

			// Send subgroup header over the stream
			// since this is a single-subgroup stream it's certain that end of group will be present in the stream
			// Extensions are present for every subgroup object within this stream, those who have no metadata to transmit MUST set their extensions length to 0
			sgh := model.NewSubGroupHeader(sub.Alias, latestGroup, model.SHWithEndOfGroup(), model.SHWithExtensions(), model.SHWithPublisherPriority(128))
			var sghBufArr [0]byte
			sghBuf := sghBufArr[:0] // Create a slice backed by the stack array

			message.EncodeSubgroupHeader(&sghBuf, sgh)
			n, err := latestStream.Write(sghBuf) // TODO: Might implement a timeout context wrapper for this line??

			if err != nil {
				fmt.Printf("Failed to write subgroup header for subscription %d, Written %d bytes out of %d-length message ,error: %v\n", sub.ID, n, len(sghBuf), err)
				pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
					RequestId:   sub.ID,
					StatusCode:  control.PublishDoneInternalError,
					StreamCount: streamCount,
					ErrorReason: model.NewReasonPhrase("Failed to write data to stream"),
				}, latestStream)
				return
			}
		}

		// send objects over latestStream
		var objIdDelta uint64
		if objIdTracker == ^uint64(0) {
			objIdDelta = obj.Location.ObjectId // 0
		} else {
			objIdDelta = obj.Location.ObjectId - objIdTracker - 1
		}
		objIdTracker = obj.Location.ObjectId

		sgo, err := model.NewSubgroupObject(objIdDelta, model.SOWithPayload(obj.Payload), model.SOWithExtensions(obj.ExtensionHeaders), model.SOWithStatus(obj.ObjectStatus))
		if err != nil {
			// Dispatcher yielded a corrupt object, close dispatcher channel, remove subscription, close all ongoing streams, send PUBLISH_DONE
			fmt.Printf("Dispatcher handed corrupt object for subscription with id: %d, error: %v\n", sub.ID, err)
			pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
				RequestId:   sub.ID,
				StatusCode:  control.PublishDoneInternalError,
				StreamCount: streamCount,
				ErrorReason: model.NewReasonPhrase("Publisher's dispatcher handed corrupt object"),
			}, latestStream)
			return
		}
		var sgoBufArr [0]byte
		sgoBuf := sgoBufArr[:0]

		message.EncodeSubgroupObject(&sgoBuf, sgo)
		n, err := latestStream.Write(sgoBuf)
		if err != nil {
			fmt.Printf("Failed to write subgroup object for subscription %d, Written %d bytes out of %d-length message ,error: %v\n", sub.ID, n, len(sgoBuf), err)
			pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
				RequestId:   sub.ID,
				StatusCode:  control.PublishDoneInternalError,
				StreamCount: streamCount,
				ErrorReason: model.NewReasonPhrase("Failed to write data to stream"),
			}, latestStream)
			return
		}
	}

	// track is over send an end of track object in a new group
	// note: ^uint64(0) + 1 = 0
	// so if the dispatcher channel is somehow closed before sending any objects EOT object is sent in group 0
	ctx, cancel := context.WithTimeout(pub.sess.Conn.Context(), openStreamTimeoutSecs)
	latestStream, err := pub.sess.Conn.OpenUniStreamSync(ctx)
	cancel()

	if err != nil {
		fmt.Printf("Failed to open stream for subscription %d, error: %v\n", sub.ID, err)
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneInternalError,
			StreamCount: streamCount,
			ErrorReason: model.NewReasonPhrase("Failed to open stream"),
		}, latestStream)
		return
	}

	streamCount++
	latestGroup++
	// Send Subgroup Header
	sgh := model.NewSubGroupHeader(sub.Alias, latestGroup, model.SHWithEndOfGroup(), model.SHWithPublisherPriority(255))
	var sghBufArr [0]byte
	sghBuf := sghBufArr[:0] // Create a slice backed by the stack array

	message.EncodeSubgroupHeader(&sghBuf, sgh)
	n, err := latestStream.Write(sghBuf) // TODO: Might implement a timeout context wrapper for this line??

	if err != nil {
		fmt.Printf("Failed to write subgroup header for subscription %d, Written %d bytes out of %d-length message ,error: %v\n", sub.ID, n, len(sghBuf), err)
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneInternalError,
			StreamCount: streamCount,
			ErrorReason: model.NewReasonPhrase("Failed to write data to stream"),
		}, latestStream)
		return
	}

	// Send EOT Object
	eotObj, _ := model.NewSubgroupObject(0, model.SOWithStatus(model.EndOfTrack))

	var sgoBufArr [0]byte
	sgoBuf := sgoBufArr[:0]

	message.EncodeSubgroupObject(&sgoBuf, eotObj)
	n, err = latestStream.Write(sgoBuf)
	if err != nil {
		fmt.Printf("Failed to write subgroup object for subscription %d, Written %d bytes out of %d-length message ,error: %v\n", sub.ID, n, len(sgoBuf), err)
		pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
			RequestId:   sub.ID,
			StatusCode:  control.PublishDoneInternalError,
			StreamCount: streamCount,
			ErrorReason: model.NewReasonPhrase("Failed to write data to stream"),
		}, latestStream)
		return
	}

	pub.cleanUpSubscription(sub, &control.PublishDoneMessage{
		RequestId:   sub.ID,
		StatusCode:  control.PublishDoneTrackEnded,
		StreamCount: streamCount,
		ErrorReason: model.NewReasonPhrase("No error - Track ended gracefully"),
	}, latestStream)
}

func (pub *Publisher) cleanUpSubscription(sub *Subscription, publishDone *control.PublishDoneMessage, latestStream transport.SendStream) {
	// 1. Close the Dispatcher channel and stop receiving any objects from it
	// 2. Close latestStream
	// 2. Send a PUBLISH_DONE message
	// 3. Remove the subscription from in-memory registry
	sub.Dispatcher.Close(sub)

	switch publishDone.StatusCode {
	case control.PublishDoneInternalError:
		latestStream.CancelWrite(quic.StreamErrorCode(quic.InternalError)) // Aborts the stream, no retransmission
	case control.PublishDoneTrackEnded:
		latestStream.Close() // Will receive on-fly objects or objects that needs retransmitting
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

// Instantiates a subscription, returns it + the latest object location
// returns ok if joined sucessfully, otherwise (with very small probability of happening) the track is over and it's dispatcher channel is closed

// Will also register the subscription to maps
func (pub *Publisher) NewSubscription(reqId uint64, ftn *model.MoqtFullTrackName, filter model.SubscriptionFilter,
	parameters []model.MoqtKeyValuePair, dispatcherChan <-chan *model.MoqtObject) (*Subscription, *model.MoqtLocation, bool) {
	pub.latestAliasMutex.Lock()
	trackAlias := pub.latestAlias
	pub.latestAlias++
	pub.latestAliasMutex.Unlock()

	sub := &Subscription{
		ID:                reqId,
		Alias:             trackAlias,
		FullTrackName:     ftn,
		Filter:            filter,
		Status:            SubscriptionStatusEstablished,
		Parameters:        parameters,
		DispatcherChannel: dispatcherChan, // Must be created by the subscribe message handler
	}

	// Get the latest object for the subscription's track and return its location
	latestObj, ok := <-sub.DispatcherChannel
	if !ok {
		sub.Status = SubscriptionStatusTerminated
		return sub, nil, false
	}

	pub.SubInNamesMutex.Lock()
	pub.ActiveIncomingSubscriptionNames[ftn.ToString()] = sub
	pub.SubInNamesMutex.Unlock()

	pub.SubInAliasesMutex.Lock()
	pub.ActiveIncomingSubscriptionAliases[trackAlias] = sub
	pub.SubInAliasesMutex.Unlock()

	return sub, &latestObj.Location, true
}

// returns false when trying to read closed dispatcher channel (means track is over before joining)
func (pub *Publisher) JoinEstablishedSubscription(sub *Subscription) bool {
	// Note: Only NextGroupStart join is implemented for now.
	switch sub.Filter.FilterType {
	case model.NextGroupStart:
		// Wait for next group start to join
		latestObj := <-sub.DispatcherChannel
		latestGroup := latestObj.Location.GroupId

		for {
			obj, ok := <-sub.DispatcherChannel
			if !ok {
				return false
			}

			if obj.Location.GroupId > latestGroup { // reached the next group, we can join now
				go pub.publishForSubscription(sub) // this will block until the track ends
				return true                        // sucessfully joined
			}
		}
	default:
		return false
	}

}

// func (pub *Publisher) HandleDrop(loc model.MoqtLocation, isGroupStart bool){

// }

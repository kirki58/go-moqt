package session

import (
	"bufio"
	"context"
	"fmt"
	"go-moq/pkg/message"
	"go-moq/pkg/model"
	"go-moq/pkg/transport"
	"io"
	"sync"
	"time"
)

const receiveNewStreamTimeout = 30 * time.Second // listen loop should receive a stream before receiveNewStreamTimeout
const waitForAliasTimeout = 5 * time.Second

type Subscriber struct {
	sess *Session

	// Active Outgoing Subscription by their Track alias
	SubOutAliasesMutex                sync.Mutex
	ActiveOutgoingSubscriptionAliases map[uint64]*Subscription

	// Active Outgoing Subscription by their Request Ids
	SubOutIDsMutex sync.Mutex
	ActiveOutgoingSubscriptionIDs map[uint64]*Subscription

	// Pending outgoing subscriptions
	SubOutPendingMutex             sync.Mutex
	PendingOutgoingSubscriptionIDs map[uint64]*Subscription // RequestId --> subscription

	// Assigned aliases to be able to find FullTrackNames from again.
	AssignedAliasesMutex sync.Mutex
	AssignedAliases      map[uint64]string

	// Used to sleep the listen data loop when there are no subscriptions.
	anySubscriptionsMu   sync.Mutex
	anySubscriptionsCond *sync.Cond
}

func NewSubscriber(sess *Session) *Subscriber {
	sub := &Subscriber{
		sess:                              sess,
		ActiveOutgoingSubscriptionAliases: make(map[uint64]*Subscription),
		ActiveOutgoingSubscriptionIDs:     make(map[uint64]*Subscription),
		AssignedAliases:                   make(map[uint64]string),
		PendingOutgoingSubscriptionIDs:    make(map[uint64]*Subscription),
	}

	sub.anySubscriptionsCond = sync.NewCond(&sub.anySubscriptionsMu)

	return sub
}

func (s *Subscriber) NewSubscription(reqId uint64, ftn *model.MoqtFullTrackName, filter model.SubscriptionFilter, parameters []model.MoqtKeyValuePair) *Subscription {
	sub := &Subscription{
		ID: reqId,
		FullTrackName: ftn,
		Filter: filter,
		Status: SubscriptionStatusPending,
		Parameters: parameters,
	}
	s.SubOutPendingMutex.Lock()
	defer s.SubOutPendingMutex.Unlock()
	s.PendingOutgoingSubscriptionIDs[reqId] = sub

	return sub
}

// Used after receiving SUBSCRIBE_OK message
func (s *Subscriber) ActivateSubscription(reqId uint64, trackAlias uint64, largestObj model.MoqtLocation) error {
	s.SubOutPendingMutex.Lock()
	defer s.SubOutPendingMutex.Unlock()
	sub, ok := s.PendingOutgoingSubscriptionIDs[reqId]
	if !ok {
		return fmt.Errorf("A pending subscription with Request ID: %d could not be found", reqId)
	}
	if sub.Status != SubscriptionStatusPending {
		return fmt.Errorf("Subscription with Request ID: %d is not in pending state, cannot be activated", reqId)
	}

	sub.Alias = trackAlias
	sub.Filter.StartLocation = largestObj
	sub.Status = SubscriptionStatusEstablished
	ch := make(chan *model.MoqtObject, 100)
	sub.ObjectSendChannel = ch // buffered channel to avoid blocking the redispatch loop, buffer size is arbitrary for now
	sub.ObjectReceiveChannel = ch

	s.SubOutAliasesMutex.Lock()
	s.ActiveOutgoingSubscriptionAliases[trackAlias] = sub
	s.SubOutAliasesMutex.Unlock()

	s.AssignedAliasesMutex.Lock()
	s.AssignedAliases[trackAlias] = sub.FullTrackName.ToString()
	s.AssignedAliasesMutex.Unlock()

	s.SubOutIDsMutex.Lock()
	s.ActiveOutgoingSubscriptionIDs[reqId] = sub
	s.SubOutIDsMutex.Unlock()

	delete(s.PendingOutgoingSubscriptionIDs, reqId)
	s.anySubscriptionsCond.Broadcast() // wake up ListenAndRedispatch if it is sleeping
	return nil
}

func (s *Subscriber) GetSubscriptionByAlias(alias uint64) (*Subscription, bool) {
	s.SubOutAliasesMutex.Lock()
	defer s.SubOutAliasesMutex.Unlock()
	sub, ok := s.ActiveOutgoingSubscriptionAliases[alias]
	return sub, ok
}

func (s *Subscriber) ListenAndRedispatch() {
	s.anySubscriptionsMu.Lock()
	for len(s.ActiveOutgoingSubscriptionAliases) == 0 {
		s.anySubscriptionsCond.Wait()
	}
	s.anySubscriptionsMu.Unlock()

	for {
		select {
		case <-s.sess.Conn.Context().Done():
			s.cleanAllSubscriptions()
			fmt.Printf("Subscriber listen streams loop exiting since session's underlying connection got closed with peer %s", s.sess.Conn.RemoteHost())
			return
		default:
			ctx, cancel := context.WithTimeout(s.sess.Conn.Context(), receiveNewStreamTimeout)
			rStream, err := s.sess.Conn.AcceptUniStream(ctx)
			cancel()
			if err != nil {
				fmt.Printf("[WARN] Could not accept new stream from peer %s: %v\n", s.sess.Conn.RemoteHost(), err)
				continue
			}

			go s.redispatchForStream(s.sess.Conn.Context(), rStream)
		}
	}
}

func (s *Subscriber) cleanAllSubscriptions(){
	s.SubOutAliasesMutex.Lock()
	for _, sub := range s.ActiveOutgoingSubscriptionAliases {
		close(sub.ObjectSendChannel)
		delete(s.ActiveOutgoingSubscriptionAliases, sub.Alias)
	}
	s.SubOutAliasesMutex.Unlock()

	s.SubOutIDsMutex.Lock()
	for id := range s.ActiveOutgoingSubscriptionIDs {
		delete(s.ActiveOutgoingSubscriptionIDs, id)
	}
	s.SubOutIDsMutex.Unlock()

	s.AssignedAliasesMutex.Lock()
	for alias := range s.AssignedAliases {
		delete(s.AssignedAliases, alias)
	}
	s.AssignedAliasesMutex.Unlock()
	
	s.SubOutPendingMutex.Lock()
	for id := range s.PendingOutgoingSubscriptionIDs {
		delete(s.PendingOutgoingSubscriptionIDs, id)
	}
	s.SubOutPendingMutex.Unlock()
}

func (s *Subscriber) TerminateSubscription(reqId uint64){
	s.SubOutIDsMutex.Lock()
	sub, ok := s.ActiveOutgoingSubscriptionIDs[reqId]
	if ok {
		delete(s.ActiveOutgoingSubscriptionIDs, reqId)
		close(sub.ObjectSendChannel)
	}
	s.SubOutIDsMutex.Unlock()

	s.SubOutAliasesMutex.Lock()
	if ok {
		delete(s.ActiveOutgoingSubscriptionAliases, sub.Alias)
	}
	s.SubOutAliasesMutex.Unlock()
	
	s.AssignedAliasesMutex.Lock()
	if ok {
		delete(s.AssignedAliases, sub.Alias)
	}
	s.AssignedAliasesMutex.Unlock()
}

func (s *Subscriber) CancelPendingSubscription(reqId uint64){
	s.SubOutPendingMutex.Lock()
	defer s.SubOutPendingMutex.Unlock()
	delete(s.PendingOutgoingSubscriptionIDs, reqId)	
}

func (s *Subscriber) redispatchForStream(ctx context.Context, rStream transport.ReceiveStream) {
	br := bufio.NewReader(rStream)
	
	receivedSubgroupHeader := false
	receivedFirstObject := false

	var streamGroup uint64                   // static
	var firstObjId uint64                    // static
	var latestObjId uint64                   // dynamic
	var forSub *Subscription                 // static
	var subgroupHeader *model.SubGroupHeader // static

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Subscriber stream listen objects loop exiting since parent context is done")
			rStream.CancelRead(0)
			return
		default:
			// First received data should be subgroup header in the stream
			if !receivedSubgroupHeader {
				sgh, err := message.DecodeSubgroupHeaderFromReader(br)
				if err != nil {
					if err == io.EOF { // received a FIN on this stream
						// assumes stream contains end of group
						return // stream ended gracefully
					}

					if terminated := s.sess.TerminateIfTerminationError(err); terminated{ // since connection context is done after this call, it will do a clean up in the parent goroutine
						continue // skip to the next iteration which will get a context done signal
					}

					fmt.Printf("Error reading subgroup header from stream: %v", err)
					return // abandon the stream
				}

				// Note for simplicity, with extensions, with end of group, with priority 128 assumed for each subgroup header

				subgroupHeader = sgh
				streamGroup = sgh.GroupId
				alias := sgh.TrackAlias

				deadline := time.Now().Add(waitForAliasTimeout)

				var sub *Subscription = nil
				for time.Now().Before(deadline){
					s.SubOutAliasesMutex.Lock()
					active, ok := s.ActiveOutgoingSubscriptionAliases[alias]
					s.SubOutAliasesMutex.Unlock()
					if ok{
						sub = active
						break
					}
				}

				if sub == nil{
					fmt.Printf("Received subgroup header for track alias %d which doesn't match any active subscription aliases, abandoning stream\n", alias)
					return // abandon the stream
				}
				forSub = sub
				receivedSubgroupHeader = true

			} else {
				sgo, err := message.DecodeSubgroupObjectFromReader(br, &subgroupHeader.SGType)
				if err != nil{
					if err == io.EOF{
						return // stream ended gracefully
					}

					if terminated := s.sess.TerminateIfTerminationError(err); terminated{
						continue // skip to the next iteration which will get a context done signal
					}

					fmt.Printf("Error reading subgroup object from stream: %v", err)
					if !receivedFirstObject{
						return
					}else{
						continue
					}
				}

				// check for end of track object
				if sgo.Status.Valid && sgo.Status.Val == model.EndOfTrack {
					return
				}

				var objRealLoc model.MoqtLocation

				if !receivedFirstObject {
					firstObjId = sgo.ObjectIdDelta
					latestObjId = firstObjId
					receivedFirstObject = true
					objRealLoc = model.MoqtLocation{GroupId: streamGroup, ObjectId: firstObjId}
				}else{
					// when sgo.ObjectIdDelta != 0:
					// since Prior Object ID Gap parsing is not implemented!
					// and In a single stream (unless) reset, QUIC guarantees in-order delivery
					// this scope is unreachable in this implementation

					latestObjId += sgo.ObjectIdDelta + 1
					objRealLoc = model.MoqtLocation{GroupId: streamGroup, ObjectId: latestObjId}
				}


				var sgid uint64
				switch subgroupHeader.SGType.SGIDMode {
				case model.SubgroupIdModeAbsentZero:
					sgid = 0
				case model.SubgroupIdModeAbsentFirstObject:
					sgid = objRealLoc.ObjectId
				case model.SubgroupIdModePresent:
					sgid = subgroupHeader.SubgroupId.Val
				}

				s.AssignedAliasesMutex.Lock()
				ftnStr, _ := s.AssignedAliases[forSub.Alias]
				s.AssignedAliasesMutex.Unlock()
				ftn, _ := model.StringToMoqtFullTrackName(ftnStr)

				// redispatch sgo
				// Normal status payload object is assumed
				obj := &model.MoqtObject{
					Location:                   objRealLoc,
					SubgroupID:                 sgid, // Multiple subgroups not implemented yet
					FullTrackName:              ftn,
					PublisherPriority:          subgroupHeader.PublisherPriority.Val, // Assumed priority present in this stream
					ObjectForwardingPreference: model.Subgroup,
					ObjectStatus:               model.Normal,
					ExtensionHeaders:           sgo.Extensions.Val, // Assumed extensions present in this stream
					Payload:                    sgo.Payload.Val,
				}
				
				pushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				select {
				case forSub.ObjectSendChannel <- obj:
				case <-pushCtx.Done():
					fmt.Printf("Timeout pushing object to ObjectSourceChannel for track %s\n", ftnStr)
					return
				}
			}
		}
	}
}

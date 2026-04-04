package data

import (
	moqt "go-moq"
	"go-moq/pkg/model"
	"go-moq/pkg/transport"
	"sync"
	"time"
)

const openStreamTimeoutSecs = 5 * time.Second

// Per-session publisher, deals with the data plane
// Publisher streams tracks to their subscribers
type Publisher struct {
	Conn transport.MOQTConnection // A reference to the session's transport-level connection in order to be able to open and close streams and send objects etc.

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
	latestGroup := ^uint64(0) // assign it to 11111.... (64) this will indicate a newly started stream
	// var latestStream transport.SendStream

	// Listen to sub.DispatcherChannel until it's open (breaks out when channel is closed by the dispatcher)
	for obj := range sub.DispatcherChannel {
		if obj.Location.GroupId > latestGroup || latestGroup == ^uint64(0) { // group boundary reached, FIN the previous stream, open a new stream
			// if latestGroup != ^uint64(0){ // Not recently joined so there is a previous stream
			// 	latestStream.Close()
			// }
			// latestGroup = obj.Location.GroupId
			// // open a new stream
			// ctx, cancel := context.WithTimeout(pub.Conn.Context(), openStreamTimeoutSecs)
			// latestStream, err := pub.Conn.OpenUniStreamSync(ctx)
			// cancel()

			// if err != nil{
			// 	// For simplicity if open stream fails we terminate the subscription right now.
			// 	sub.Dispatcher.Close(sub)
			// }

			// // Send subgroup header over the stream
			// // since this is a single-subgroup stream it's certain that end of group will be present in the stream
			// // Extensions are present for every subgroup object within this stream, those who have no metadata to transmit MUST set their extensions length to 0
			// sgh := model.NewSubGroupHeader(sub.Alias, latestGroup, model.SHWithEndOfGroup(), model.SHWithExtensions(), model.SHWithPublisherPriority(128))
			// var buf [32]byte
			// sghBuf := buf[:0] // Create a slice backed by the stack array

			// message.EncodeSubgroupHeader(&sghBuf, sgh)
			// n, err := latestStream.Write(sghBuf)
		}

		// send objects over latestStream
		// obj.ToSubgroupObject
		// buf = encode object
		// latestStream.Write(buf)
	}

	// track is over send an end of track object
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

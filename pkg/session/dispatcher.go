package session

import (
	"fmt"
	"go-moq/pkg/model"
	"sync"
)

type Dispatcher struct {
	FullTrackName *model.MoqtFullTrackName // The track we are encoding

	SubChannelsMu   sync.Mutex
	SubChannels     map[*Subscription]chan<- *model.MoqtObject // Publishers-->ActiveIncomingSubscriptionNames['FullTrackName'] gives the subscription , BUFFERED send-only channel !!!
	SubDropChannels map[*Subscription]chan<- struct{}          // Non-buffered notification channel, dispatcher uses this to inform the publisher that it needs to drop some objects
}

func NewDispatcher(ftn *model.MoqtFullTrackName) (*Dispatcher){
	return &Dispatcher{
		FullTrackName: ftn,
		SubChannels: make(map[*Subscription]chan<- *model.MoqtObject),
		SubDropChannels: make(map[*Subscription]chan<- struct{}),
	}
}

// Dispatch this object to all publishers over channels (used by the encoder)
func (d *Dispatcher) Dispatch(obj *model.MoqtObject) {
	for sub, ch := range d.SubChannels {
		// Send the object to the publisher's channel
		select {
		case ch <- obj:
		default:
			// Buffer full, handle overflow (drop and log)
			fmt.Printf("Warning: Couldn't push object over channel, either publisher buffer is full or the channel is closed for subscription id: %d, sending a drop notification to subscription's publisher\n", sub.ID)
			// Notify the publisher to drop objects
			dropCh := d.SubDropChannels[sub]
			select {
			case dropCh <- struct{}{}:
			default:
				// Drop notification already pending
			}
			continue
		}
	}
}

// Create and register send channels, return their receive counterparts
func (d *Dispatcher) RegisterNewSubChannel(sub *Subscription, bufSize uint) (<-chan *model.MoqtObject, <-chan struct{}) {
	// Note: channel directions will implicitly convert, no worries!

	// Create a buffered channel for objects
	// and a size-1 channel for drop notifications to the publisher
	objCh := make(chan *model.MoqtObject, bufSize)
	dropCh := make(chan struct{}, 1)

	// Register the channels in the dispatcher
	d.SubChannelsMu.Lock()
	defer d.SubChannelsMu.Unlock()
	d.SubChannels[sub] = objCh
	d.SubDropChannels[sub] = dropCh

	return objCh, dropCh
}

// Publisher decided to close it's channel received an unsubscribe, publisher's remote peer terminated the transport connectic etc.
func (d *Dispatcher) Close(sub *Subscription) {
	d.SubChannelsMu.Lock()
	defer d.SubChannelsMu.Unlock()
	ch := d.SubChannels[sub]
	delete(d.SubChannels, sub)
	close(ch)

	dropCh := d.SubDropChannels[sub]
	delete(d.SubDropChannels, sub)
	close(dropCh)
}

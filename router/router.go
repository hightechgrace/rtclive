package router

import (
	"sync"

	"github.com/gofrs/uuid"
	mediaserver "github.com/notedit/media-server-go"
	"github.com/notedit/media-server-go/sdp"
)

type Publisher struct {
	id         string
	incoming   *mediaserver.IncomingStream
	videotrack *mediaserver.IncomingStreamTrack
	audiotrack *mediaserver.IncomingStreamTrack
	transport  *mediaserver.Transport
}

func NewPublisher(incoming *mediaserver.IncomingStream, transport *mediaserver.Transport) *Publisher {

	var videoTrack *mediaserver.IncomingStreamTrack
	var audioTrack *mediaserver.IncomingStreamTrack

	if len(incoming.GetVideoTracks()) > 0 {
		videoTrack = incoming.GetVideoTracks()[0]
	}

	if len(incoming.GetAudioTracks()) > 0 {
		audioTrack = incoming.GetAudioTracks()[0]
	}

	publisher := &Publisher{
		id:         incoming.GetID(),
		incoming:   incoming,
		videotrack: videoTrack,
		audiotrack: audioTrack,
		transport:  transport,
	}
	return publisher
}

func NewPublisherWithID(ID string, videotrack *mediaserver.IncomingStreamTrack, audiotrack *mediaserver.IncomingStreamTrack) *Publisher {

	publisher := &Publisher{
		id:         ID,
		videotrack: videotrack,
		audiotrack: audiotrack,
	}
	return publisher
}

func (p *Publisher) GetID() string {
	return p.id
}

func (p *Publisher) GetStream() *mediaserver.IncomingStream {
	return p.incoming
}

func (p *Publisher) GetVideoTrack() *mediaserver.IncomingStreamTrack {
	return p.videotrack
}

func (p *Publisher) GetAudioTrack() *mediaserver.IncomingStreamTrack {
	return p.audiotrack
}

func (p *Publisher) GetTransport() *mediaserver.Transport {
	return p.transport
}

func (p *Publisher) Stop() {

	if p.incoming != nil {
		p.incoming.Stop()
	}

	if p.transport != nil {
		p.transport.Stop()
	}

}

type Subscriber struct {
	id          string
	publisherId string
	outgoing    *mediaserver.OutgoingStream
	transport   *mediaserver.Transport
}

func (s *Subscriber) GetID() string {
	return s.id
}

func (s *Subscriber) GetPublisherID() string {
	return s.publisherId
}

func (s *Subscriber) GetStream() *mediaserver.OutgoingStream {
	return s.outgoing
}

func (s *Subscriber) GetTransport() *mediaserver.Transport {
	return s.transport
}

func (s *Subscriber) Stop() {
	s.outgoing.Stop()
	s.transport.Stop()
}

type MediaRouter struct {
	routerID     string
	capabilities map[string]*sdp.Capability
	endpoint     *mediaserver.Endpoint
	publisher    *Publisher
	subscribers  map[string]*Subscriber
	originUrl    string
	origin       bool
	sync.Mutex
}

func NewMediaRouter(routerID string, endpoint *mediaserver.Endpoint, capabilities map[string]*sdp.Capability, origin bool) *MediaRouter {
	router := &MediaRouter{}
	router.routerID = routerID
	router.endpoint = endpoint
	router.capabilities = capabilities
	router.origin = origin

	router.subscribers = make(map[string]*Subscriber)
	return router
}

func (r *MediaRouter) GetID() string {
	return r.routerID
}

func (r *MediaRouter) IsOrgin() bool {
	return r.origin
}

func (r *MediaRouter) GetPublisher() *Publisher {
	return r.publisher
}

func (r *MediaRouter) SetPublisher(publisher *Publisher) {
	r.publisher = publisher
}

func (r *MediaRouter) SetOriginUrl(origin string) {
	r.originUrl = origin
}

func (s *MediaRouter) GetOriginUrl() string {
	return s.originUrl
}

func (s *MediaRouter) GetSubscribers() map[string]*Subscriber {
	return s.subscribers
}

func (r *MediaRouter) CreatePublisher(sdpStr string) (*Publisher, string) {
	offer, err := sdp.Parse(sdpStr)
	if err != nil {
		panic(err)
	}

	transport := r.endpoint.CreateTransport(offer, nil)
	transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

	answer := offer.Answer(transport.GetLocalICEInfo(),
		transport.GetLocalDTLSInfo(),
		r.endpoint.GetLocalCandidates(),
		r.capabilities)

	transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

	streamInfo := offer.GetFirstStream()
	incoming := transport.CreateIncomingStream(streamInfo)

	var videoTrack *mediaserver.IncomingStreamTrack
	var audioTrack *mediaserver.IncomingStreamTrack

	if len(incoming.GetVideoTracks()) > 0 {
		videoTrack = incoming.GetVideoTracks()[0]
	}

	if len(incoming.GetAudioTracks()) > 0 {
		audioTrack = incoming.GetAudioTracks()[0]
	}

	r.publisher = &Publisher{
		id:         streamInfo.GetID(),
		incoming:   incoming,
		videotrack: videoTrack,
		audiotrack: audioTrack,
		transport:  transport,
	}

	return r.publisher, answer.String()
}

func (r *MediaRouter) CreateSubscriber(sdpStr string) (*Subscriber, string) {
	offer, err := sdp.Parse(sdpStr)
	if err != nil {
		panic(err)
	}

	transport := r.endpoint.CreateTransport(offer, nil)
	transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

	answer := offer.Answer(transport.GetLocalICEInfo(),
		transport.GetLocalDTLSInfo(),
		r.endpoint.GetLocalCandidates(),
		r.capabilities)

	transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

	subId := uuid.Must(uuid.NewV4()).String()

	audio := r.publisher.audiotrack != nil
	video := r.publisher.videotrack != nil

	outgoing := transport.CreateOutgoingStreamWithID(subId, audio, video)

	if audio {
		outgoing.GetAudioTracks()[0].AttachTo(r.publisher.audiotrack)
	}

	if video {
		outgoing.GetVideoTracks()[0].AttachTo(r.publisher.videotrack)
	}

	subscriber := &Subscriber{
		id:          subId,
		publisherId: r.publisher.GetID(),
		outgoing:    outgoing,
		transport:   transport,
	}

	r.Lock()
	r.subscribers[subId] = subscriber
	r.Unlock()

	answer.AddStream(outgoing.GetStreamInfo())

	return subscriber, answer.String()
}

func (r *MediaRouter) StopSubscriber(subscriberId string) {
	subscriber := r.subscribers[subscriberId]
	if subscriber == nil {
		return
	}

	subscriber.Stop()

	r.Lock()
	delete(r.subscribers, subscriberId)
	r.Unlock()
}

func (r *MediaRouter) Stop() {
	r.Lock()
	defer r.Unlock()
	if r.publisher != nil {
		r.publisher.Stop()
	}

	for _, subscriber := range r.subscribers {
		subscriber.Stop()
	}

	r.publisher = nil
	r.subscribers = nil
}
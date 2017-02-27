package chrome

var eventConstructors = map[string]func()(interface{}){
	"Page.loadEventFired":       func() interface{} { return new(PageLoadEventFired) },
	"Network.loadingFailed":     func() interface{} { return new(NetworkLoadingFailed) },
	"Network.loadingFinished":   func() interface{} { return new(NetworkLoadingFinished) },
	"Network.requestWillBeSent": func() interface{} { return new(NetworkRequestWillBeSent) },
	"Network.responseReceived":  func() interface{} { return new(NetworkResponseReceived) },
}

type PageLoadEventFired struct {
	timestamp float64
}

type NetworkLoadingFailed struct {
	RequestId string
}

type NetworkLoadingFinished struct {
	RequestId string
}

type NetworkRequest struct {
	Url string
	Method string
	Headers map[string]string
	MixedContentType string
	InitialPriority string
	ReferrerPolicy string
}

type NetworkRequestWillBeSent struct{
	RequestId string
	FrameId string
	LoaderId string
	DocumentURL string
	Request *NetworkRequest
	Timestamp float64
	WallTime float64
	Type string
}

type NetworkResponse struct {
	Url string
	Status int32
	StatusText string
	Headers map[string]string
	HeadersText string
	MimeType string
	RequestHeaders map[string]string
	RequestHeadersText string
	RemoteIPAddress string
	Protocol string
}

type NetworkResponseReceived struct {
	RequestId string
	Response *NetworkResponse
}
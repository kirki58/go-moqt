package control

// Note, [Cite: Section 9.3]: To ensure future extensibility of MOQT, endpoints MUST ignore unknown setup parameters.

// Setup Parameter IDs (Section 9.3.1)
// Constant across all MOQT versions
const (
	SetupParamPath                  = 0x01 // [cite: 925]
	SetupParamMaxRequestID          = 0x02 // [cite: 932]
	SetupParamAuthToken             = 0x03 // [cite: 938]
	SetupParamMaxAuthTokenCacheSize = 0x04 // [cite: 934]
	SetupParamAuthority             = 0x05 // [cite: 920]
	SetupParamMoqtImplementation    = 0x07 // [cite: 943]
)

// Version-Specific Parameter IDs (Section 9.2.1)
// Valid only for the negotiated version (Draft-15) [cite: 797]
const (
	ParamDeliveryTimeout    = 0x02 // [cite: 845]
	ParamAuthToken          = 0x03 // [cite: 800]
	ParamMaxCacheDuration   = 0x04 // [cite: 857]
	ParamExpires            = 0x08 // [cite: 884]
	ParamLargestObject      = 0x09 // [cite: 893]
	ParamPublisherPriority  = 0x0E // [cite: 863]
	ParamForward            = 0x10 // [cite: 894]
	ParamSubscriberPriority = 0x20 // [cite: 868]
	ParamSubscriptionFilter = 0x21 // [cite: 880]
	ParamGroupOrder         = 0x22 // [cite: 873]
	ParamDynamicGroups      = 0x30 // [cite: 896]
	ParamNewGroupRequest    = 0x32 // [cite: 900]
)

package model

// Layer represents a single visual layer/sprite in the PNGTuber avatar.
type Layer struct {
	Identification int64    `json:"identification"`
	ParentID       *int64   `json:"parentId"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	ZIndex         int      `json:"zindex"`
	Pos            Vector2  `json:"pos"`
	Offset         Vector2  `json:"offset"`
	Frames         int      `json:"frames"`
	AnimSpeed      float32  `json:"animSpeed"`
	Clipped        bool     `json:"clipped"`
	StretchAmount  float32  `json:"stretchAmount"`
	RLimitMin      float32  `json:"rLimitMin"`
	RLimitMax      float32  `json:"rLimitMax"`
	RotDrag        float32  `json:"rotDrag"`
	Drag           float32  `json:"drag"`
	XAmp           float32  `json:"xAmp"`
	XFrq           float32  `json:"xFrq"`
	YAmp           float32  `json:"yAmp"`
	YFrq           float32  `json:"yFrq"`
	IgnoreBounce   bool     `json:"ignoreBounce"`
	ShowBlink      int      `json:"showBlink"` // 0: always, 1: only blinking, 2: only not blinking
	ShowTalk       int      `json:"showTalk"`  // 0: always, 1: only talking, 2: only not talking
	CostumeLayers  [10]int  `json:"costumeLayers"`
	ImageData      []byte   `json:"-"`
	ImageWidth     int      `json:"-"`
	ImageHeight    int      `json:"-"`
}

// NewDefaultLayer creates a layer with standard default values.
func NewDefaultLayer(id int64) *Layer {
	return &Layer{
		Identification: id,
		ParentID:       nil,
		Type:           "sprite",
		Frames:         1,
		AnimSpeed:      0,
		RLimitMin:      -180,
		RLimitMax:      180,
		CostumeLayers:  [10]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	}
}

// IsVisible determines whether this layer should be rendered based on:
// - costume: 1-indexed costume slot (1 to 10)
// - isBlinking: whether the avatar is currently blinking
// - isTalking: whether the avatar is currently speaking (audio threshold exceeded)
func (l *Layer) IsVisible(costume int, isBlinking bool, isTalking bool) bool {
	// 1. Costume visibility
	if costume >= 1 && costume <= 10 {
		if l.CostumeLayers[costume-1] == 0 {
			return false
		}
	}

	// 2. Blink visibility (PNGTuber-Plus: 0=Always, 1=Eyes Open/Not Blinking, 2=Blinking/Closed Eyes)
	switch l.ShowBlink {
	case 1: // Visible ONLY when NOT blinking (Eyes Open / Normal)
		if isBlinking {
			return false
		}
	case 2: // Visible ONLY when BLINKING (Eyes Closed)
		if !isBlinking {
			return false
		}
	}

	// 3. Talk visibility (PNGTuber-Plus: 0=Always, 1=Quiet/Muted, 2=Talking/Speaking)
	switch l.ShowTalk {
	case 1: // Visible ONLY when NOT talking (Quiet / Closed Mouth)
		if isTalking {
			return false
		}
	case 2: // Visible ONLY when TALKING (Speaking / Open Mouth)
		if !isTalking {
			return false
		}
	}

	return true
}

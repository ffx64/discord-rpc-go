package client

type ActivityType int

const (
	Playing   ActivityType = 0
	Listening ActivityType = 2
	Watching  ActivityType = 3
	Competing ActivityType = 5
)

type Button struct {
	Label string `json:"label"`
	Url   string `json:"url"`
}

type Party struct {
	ID   string `json:"id,omitempty"`
	Size []int  `json:"size,omitempty"` // [current, max]
}

type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

type Timestamps struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

type Activity struct {
	Type       ActivityType      `json:"type,omitempty"`
	State      string            `json:"state,omitempty"`
	Details    string            `json:"details,omitempty"`
	Timestamps *Timestamps       `json:"timestamps,omitempty"`
	Assets     *Assets           `json:"assets,omitempty"`
	Party      *Party            `json:"party,omitempty"`
	Secrets    map[string]string `json:"secrets,omitempty"`
	Buttons    []Button          `json:"buttons,omitempty"`
}

func (a Activity) IsEmpty() bool {
	return a.State == "" &&
		a.Details == "" &&
		a.Timestamps == nil &&
		a.Assets == nil &&
		a.Party == nil &&
		len(a.Secrets) == 0 &&
		len(a.Buttons) == 0
}

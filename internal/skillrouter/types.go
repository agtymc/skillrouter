package skillrouter

const AppVersion = "0.5.2b"

type Preset struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Config struct {
	Presets []Preset `json:"presets"`
}

type SkillItem struct {
	Label string
	Path  string
}

package discord

const Type = "discord"

type Notifier struct {
	Webhook string `json:"webhook"`
}

type Payload struct {
	Title   string   `json:"username"`
	Content string   `json:"content"`
	Avatar  string   `json:"avatar_url"`
	Embeds  []*Embed `json:"embeds"`
}

func (p *Payload) AddEmbed(embed *Embed) {
	p.Embeds = append(p.Embeds, embed)
}

type Embed struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Color       int      `json:"color"`
	Fields      []*Field `json:"fields"`
}

func (e *Embed) AddField(field *Field) {
	e.Fields = append(e.Fields, field)
}

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

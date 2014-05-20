package contextual

// Content is a string-valued data.Value that is known to be a certain content type.
type Content struct {
	data.String
	Kind ContentKind
}

type ContentKind int

const (
	HTML ContentKind = iota
	JS
	JSStr
	URI
	Attr
	CSS
	Text
)

var kindAttrToContentKind = map[string]ContentKind{
	"attributes": Attr,
	"css":        CSS,
	"html":       HTML,
	"js":         JS,
	"text":       Text,
	"uri":        URI,
}

var kindAttrToState = map[string]state{
	"attributes": stateAttr,
	"css":        stateCSS,
	"html":       stateText,
	"js":         stateJS,
	"text":       stateText,
	"uri":        stateURL,
}

// PrintDirective is a print directive that identifies the type of content that
// it consumes and produces.
type PrintDirective interface {
	ContentKind() ContentKind
}

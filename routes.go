package nues

type RouteResponse map[string]any
type RouteCallType int
type Routes map[string]Route

const (
	COMMAND RouteCallType = iota
	QUERY
	HANDLER
)

type Route struct {
	name    string
	public  bool
	call    RouteCallType
	handler func() any
}

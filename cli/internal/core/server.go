package core

type ResourceRef struct {
	Type string
	ID   string
	Name string
}

type Server struct {
	ID                  string
	Provider            string
	Name                string
	PublicIP            string
	AssociatedResources []ResourceRef
}

func (s *Server) IP() string {
	if s == nil {
		return ""
	}

	return s.PublicIP
}

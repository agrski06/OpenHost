package core

type Server struct {
	ID       string
	Provider string
	Name     string
	PublicIP string
}

func (s *Server) IP() string {
	if s == nil {
		return ""
	}

	return s.PublicIP
}

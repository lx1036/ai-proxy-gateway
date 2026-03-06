package bootstrap

import "fmt"

type Server struct {
	configController model.ConfigStoreController
	ConfigStores     []model.ConfigStoreController
}

func (s *Server) initControllers(args *PilotArgs) error {

	if err := s.initConfigController(args); err != nil {
		return fmt.Errorf("error initializing config controller: %v", err)
	}
	if err := s.initServiceController(args); err != nil {
		return fmt.Errorf("error initializing service controllers: %v", err)
	}

	return nil

}

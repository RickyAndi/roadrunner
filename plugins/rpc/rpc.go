package rpc

import (
	"github.com/spiral/endure/errors"
	"github.com/spiral/goridge/v2"
	"github.com/spiral/roadrunner/v2/plugins/config"
	"net/rpc"
)

type RPCService interface {
	Name() string
	RPCService() (interface{}, error)
}

// ServiceName contains default service name.
const ServiceName = "rpc"

type services struct {
	service interface{}
	name    string
}

// Service is RPC service.
type Service struct {
	rpc      *rpc.Server
	services []services
	config   Config
	close    chan struct{}
}

// Init rpc service. Must return true if service is enabled.
func (s *Service) Init(cfg config.Provider) error {
	if !cfg.Has(ServiceName) {
		return errors.E(errors.Disabled)
	}

	err := cfg.UnmarshalKey(ServiceName, &s.config)
	if err != nil {
		return err
	}
	s.config.InitDefaults()

	if s.config.Disabled {
		return errors.E(errors.Disabled)
	}

	return s.config.Valid()
}

// Serve serves the service.
func (s *Service) Serve() chan error {
	s.close = make(chan struct{}, 1)
	errCh := make(chan error, 1)

	s.rpc = rpc.NewServer()

	// Attach all services
	for i := 0; i < len(s.services); i++ {
		err := s.Register(s.services[i].name, s.services[i].service)
		if err != nil {
			errCh <- errors.E(errors.Op("register service"), err)
			return errCh
		}
	}

	ln, err := s.config.Listener()
	if err != nil {
		errCh <- err
		return errCh
	}

	go func() {
		for {
			select {
			case <-s.close:
				// log error
				errCh <- ln.Close()
				return
			default:
				conn, err := ln.Accept()
				if err != nil {
					continue
				}

				go s.rpc.ServeCodec(goridge.NewCodec(conn))
			}
		}
	}()

	return errCh
}

// Stop stops the service.
func (s *Service) Stop() error {
	s.close <- struct{}{}
	return nil
}

func (s *Service) Depends() []interface{} {
	return []interface{}{
		s.RegisterService,
	}
}

func (s *Service) RegisterService(p RPCService) error {
	service, err := p.RPCService()
	if err != nil {
		return err
	}

	s.services = append(s.services, services{
		service: service,
		name:    p.Name(),
	})
	return nil
}

// Register publishes in the server the set of methods of the
// receiver value that satisfy the following conditions:
//	- exported method of exported type
//	- two arguments, both of exported type
//	- the second argument is a pointer
//	- one return value, of type error
// It returns an error if the receiver is not an exported type or has
// no suitable methods. It also logs the error using package log.
func (s *Service) Register(name string, svc interface{}) error {
	if s.rpc == nil {
		return errors.E("RPC service is not configured")
	}

	return s.rpc.RegisterName(name, svc)
}

// Client creates new RPC client.
func (s *Service) Client() (*rpc.Client, error) {
	conn, err := s.config.Dialer()
	if err != nil {
		return nil, err
	}

	return rpc.NewClientWithCodec(goridge.NewClientCodec(conn)), nil
}

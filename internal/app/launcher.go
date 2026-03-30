package app

import "portcut/internal/platform"

type ServiceFactory func() (platform.Service, error)

type ProgramRunner func() error

type ProgramFactory func(Workflow, platform.Capabilities) ProgramRunner

type Launcher struct {
	serviceFactory ServiceFactory
	programFactory ProgramFactory
}

func NewLauncher(serviceFactory ServiceFactory, programFactory ProgramFactory) Launcher {
	if serviceFactory == nil {
		serviceFactory = platform.NewService
	}

	return Launcher{
		serviceFactory: serviceFactory,
		programFactory: programFactory,
	}
}

func (l Launcher) Run() error {
	service, err := l.serviceFactory()
	if err != nil {
		return err
	}

	runner := l.programFactory(NewWorkflow(service), service.Capabilities())
	if runner == nil {
		return nil
	}

	return runner()
}

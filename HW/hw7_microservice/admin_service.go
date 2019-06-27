package main

import "io"

type AdminServerImpl struct {
}

func (as *AdminServerImpl) Logging(_ *Nothing, out Admin_LoggingServer) error {
	for {
		event := Event{
			Timestamp: 0,
			Consumer:  "",
			Method:    "",
			Host:      "",
		}
		err := out.Send(&event)
		if err != nil {
			if err == io.EOF {
				return nil
			} else {
				return nil
			}
		}
	}
}

func (as *AdminServerImpl) Statistics(*StatInterval, Admin_StatisticsServer) error {
	panic("implement me")
}

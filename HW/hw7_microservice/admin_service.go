package main

/*type AdminServerImpl struct {
	Streams *[] *Stream
	mu      *sync.RWMutex
}

type Stream struct {
	Channel chan *Event
	Ctx     context.Context
}

func (as *AdminServerImpl) Close() {
	for _, stream := range *as.Streams {
		close(stream.Channel)
	}
}

func (as *AdminServerImpl) addStream(ctx context.Context) *Stream {
	as.mu.Lock()
	defer as.mu.Unlock()
	stream := &Stream{
		Channel: make(chan *Event),
		Ctx:     ctx,
	}
	*as.Streams = append(*as.Streams, stream)
	return stream
}

func (as *AdminServerImpl) GetStream(ctx context.Context) (*Stream, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	stream := as.addStream(ctx)
	return stream, cancel
}

func (as *AdminServerImpl) Logging(_ *Nothing, out Admin_LoggingServer) error {
	stream, cancelFunc := as.GetStream(out.Context())
	defer cancelFunc()
	for ev := range stream.Channel {
		err := out.Send(ev)
		if err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}
	}
	return nil
}

func (as *AdminServerImpl) Statistics(st *StatInterval, out Admin_StatisticsServer) error {
	stream, cancelFunc := as.GetStream(out.Context())
	defer cancelFunc()

	mu := &sync.Mutex{}
	evs := make([]*Event, 0, 8)

	go func() {
		for ev := range stream.Channel {
			mu.Lock()
			evs = append(evs, ev)
			mu.Unlock()
		}
	}()

	ticker := time.NewTicker(time.Duration(st.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for tt := range ticker.C {
		mu.Lock()
		stat := countStatistic(evs)
		stat.Timestamp = tt.Unix()
		evs = make([]*Event, 0, 8)
		mu.Unlock()
		err := out.Send(stat)
		if err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}
	}
	return nil
}

func countStatistic(evs []*Event) *Stat {

	byMethod := make(map[string]uint64, 4)
	byConsumer := make(map[string]uint64, 4)

	for _, ev := range evs {
		if _, ok := byMethod[ev.Method]; ok {
			byMethod[ev.Method]++
		} else {
			byMethod[ev.Method] = 1
		}

		if _, ok := byConsumer[ev.Consumer]; ok {
			byConsumer[ev.Consumer]++
		} else {
			byConsumer[ev.Consumer] = 1
		}
	}

	return &Stat{
		Timestamp:  0,
		ByMethod:   byMethod,
		ByConsumer: byConsumer,
	}
}*/

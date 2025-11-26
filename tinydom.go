package tinydom

type DOM struct {
	log func(v ...any)
}

func New(log func(v ...any)) *DOM {

	t := &DOM{
		log: log,
	}

	return t
}

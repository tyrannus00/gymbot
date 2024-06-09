package exercises

type ex int

const (
	BENCH ex = iota
	SQUAT
	DEADLIFT
)

type Exercise interface {
	Ex() ex
}

func (e ex) Ex() ex {
	return e
}

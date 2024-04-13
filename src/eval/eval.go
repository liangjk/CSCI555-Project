package main

const times = 5

func main() {
	failureParallel()
	failureContend()
	for i := 0; i < times; i++ {
		oneClientSpeed(i)
		manyClientSpeed(i)
		manyClientContend(i)
		clientFailure(i)
	}
}

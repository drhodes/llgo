package main

func main() {
	m := make(map[int]int)
	delete(m, 0) // no-op
	m[0] = 1
	println(len(m))
	delete(m, 1) // no-op
	println(len(m), m[0])
	delete(m, 0) // delete element in map
	println(len(m), m[0])
}

package util

func GCD(x, y int) int {
	tmp := x % y
	if tmp > 0 {
		return GCD(y, tmp)
	} else {
		return y
	}
}

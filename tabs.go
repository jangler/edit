package edit

var (
	p         []rune              // Reuse the same array
	tabUnlock = make(chan int, 1) // And use a channel as mutex
)

func init() {
	tabUnlock <- 1
}

// Expand tabs to spaces; tabWidth must be > 0
func expand(s []rune, tabWidth int) []rune {
	<-tabUnlock
	if len(p) < tabWidth*len(s) {
		p = make([]rune, tabWidth*len(s))
	}
	col := 0

	for _, ch := range s {
		if ch == '\t' {
			p[col] = ' '
			col++
			for col%tabWidth != 0 {
				p[col] = ' '
				col++
			}
		} else {
			p[col] = ch
			col++
		}
	}

	s = p[:col]
	tabUnlock <- 1
	return s
}

// Return width of expanded string in columns; tabWidth must be > 0
func columns(s []rune, tabWidth int) int {
	col := 0
	for _, ch := range s {
		if ch == '\t' {
			col += tabWidth - col%tabWidth
		} else {
			col++
		}
	}
	return col
}

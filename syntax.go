package edit

import "regexp"

const noneTag = -1

// Rule is a rule for matching syntax and applying a tag to it.
type Rule struct {
	re  *regexp.Regexp
	tag int
}

// NewRule returns an initialized Rule by compiling pattern into a regular
// expression, or returns an error if pattern fails to compile.
func NewRule(pattern string, tag int) (Rule, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return Rule{}, err
	}
	return Rule{re, tag}, nil
}

// Fragment is a string annotated with a tag.
type Fragment struct {
	Text string
	Tag  int
}

type syntax []Rule

func (rules syntax) split(s string) <-chan Fragment {
	c := make(chan Fragment)
	go func() {
		if s == "" {
			c <- Fragment{s, noneTag}
		}
		for s != "" {
			// Find the first matching rule
			var minLoc []int
			var minRule Rule
			for _, rule := range rules {
				loc := rule.re.FindStringIndex(s)
				if loc != nil && (minLoc == nil || loc[0] < minLoc[0]) {
					minLoc = loc
					minRule = rule
				}
			}
			// Send fragments
			if minLoc == nil {
				c <- Fragment{s, noneTag}
				s = ""
			} else {
				if minLoc[0] > 0 {
					c <- Fragment{s[:minLoc[0]], noneTag}
				}
				c <- Fragment{s[minLoc[0]:minLoc[1]], minRule.tag}
				s = s[minLoc[1]:]
			}
		}
		close(c)
	}()
	return c
}

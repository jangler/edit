package edit

import "regexp"

const noneTag = -1

// Rule is a rule for matching syntax and applying a tag to it.
type Rule struct {
	start, end *regexp.Regexp
	tag        int
}

// NewRule returns an initialized Rule by compiling begin and end into regular
// expressions, or returns an error if either expression fails to compile. If
// end is an empty string, the rule only matches begin.
func NewRule(begin, end string, tag int) (Rule, error) {
	beginRegexp, err := regexp.Compile(begin)
	if err != nil {
		return Rule{}, err
	}
	if end != "" {
		endRegexp, err := regexp.Compile(end)
		if err != nil {
			return Rule{}, err
		}
		return Rule{beginRegexp, endRegexp, tag}, nil
	}
	return Rule{beginRegexp, nil, tag}, nil
}

type Fragment struct {
	text string
	tag  int
}

type syntax []Rule

func (rules syntax) split(s string) <-chan Fragment {
	c := make(chan Fragment)
	go func() {
		for s != "" {
			// Find the first matching rule
			var minLoc []int
			var minRule Rule
			for _, rule := range rules {
				loc := rule.start.FindStringIndex(s)
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
				if minRule.end == nil {
					c <- Fragment{s[minLoc[0]:minLoc[1]], minRule.tag}
					s = s[minLoc[1]:]
				} else {
					endLoc := minRule.end.FindStringIndex(s[minLoc[1]:])
					if endLoc == nil {
						c <- Fragment{s, noneTag}
						s = ""
					} else {
						endIndex := minLoc[1] + endLoc[1]
						c <- Fragment{s[minLoc[0]:endIndex], minRule.tag}
						s = s[endIndex:]
					}
				}
			}
		}
		close(c)
	}()
	return c
}

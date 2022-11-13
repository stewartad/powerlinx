package powerlinx

import "sort"

func (s *Site) sortAllPages() []Page {
	all := make([]Page, 0, len(s.PageMap))
	for _, value := range s.PageMap {
		all = append(all, value)
	}
	sort.Sort(byTime(all))
	return all
}

func sortPageList(pages []Page) []Page {
	sort.Sort(byTime(pages))
	return pages
}

// Create Sort Interface for Pages
type byTime []Page

func (t byTime) Len() int {
	return len(t)
}

func (t byTime) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t byTime) Less(i, j int) bool {
	return getDate(t[j]).Before(getDate(t[i]))
}

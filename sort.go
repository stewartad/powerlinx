package powerlinx

import "sort"

func (s *Site) sortAllPages() []*DetailPage {
	all := make([]*DetailPage, 0, len(s.PageMap))
	for _, value := range s.PageMap {
		all = append(all, value)
	}
	sort.Sort(byTime(all))
	return all
}

func sortPageList(pages []*DetailPage) []*DetailPage {
	sort.Sort(byTime(pages))
	return pages
}

// Create Sort Interface for Pages
type byTime []*DetailPage

func (t byTime) Len() int {
	return len(t)
}

func (t byTime) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t byTime) Less(i, j int) bool {
	return t[j].CreatedAt.Before(t[i].CreatedAt)
}

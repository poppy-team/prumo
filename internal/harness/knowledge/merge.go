package knowledge

// MergeFrom folds another store into this one for derived projections. Records
// already present are kept as-is: a later run never overwrites the knowledge an
// earlier run established, it only adds what is missing. Locator bindings merge
// the same way, so identity accumulated across runs is preserved (W3.7).
func (s *Store) MergeFrom(other *Store) {
	if other == nil {
		return
	}
	records, rels := other.Snapshot()
	for _, r := range records {
		if _, exists := s.records[r.ID]; exists {
			continue
		}
		s.records[r.ID] = r
	}
	for _, rel := range rels {
		s.rels = append(s.rels, rel)
	}
	if s.aliases == nil {
		s.aliases = map[string]string{}
	}
	for locator, id := range other.aliases {
		if _, exists := s.aliases[locator]; !exists {
			s.aliases[locator] = id
		}
	}
}

// Merge folds several stores into one derived view, deterministically.
func Merge(stores ...*Store) *Store {
	merged := New()
	for _, s := range stores {
		merged.MergeFrom(s)
	}
	return merged
}

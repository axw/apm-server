package model

type mixinMetadata struct {
	fieldLabels []Label
}

// SetLabels sets the labels on the metadata object.
func (m *Metadata) SetLabels(labels ...Label) {
	m.pb.Labels = m.pb.Labels[:0]
	m.fieldLabels = append(m.fieldLabels[:0], labels...)
	for i := range m.fieldLabels {
		m.pb.Labels[i] = &m.fieldLabels[i].pb
	}
}

// Labels returns the labels in m.
func (m *Metadata) Labels() []Label {
	return m.fieldLabels
}

type mixinTransactionContext struct {
	fieldCustom       keyValueList
	fieldExperimental Any
}

// SetCustom sets custom transaction context.
func (t *TransactionContext) SetCustom(kv ...KeyValue) {
	t.fieldCustom.set(kv...)
	t.pb.Custom = t.fieldCustom.pb.Values
}

// SetExperimental sets experimental transaction context.
func (t *TransactionContext) SetExperimental(v Any) {
	t.fieldExperimental = v
	t.pb.Experimental = &t.fieldExperimental.pb
}

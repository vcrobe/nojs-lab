package modal

// ModalType defines the visual style of the dialog.
type ModalType int

const (
	Information ModalType = iota // Default
	Warning
	Error
)

// ModalResult tells the parent *which* button was clicked.
type ModalResult int

const (
	Ok ModalResult = iota
	Cancel
)

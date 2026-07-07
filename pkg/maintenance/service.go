package maintenance

type Service interface {
	Activate(title string, text string)
	Deactivate()
}

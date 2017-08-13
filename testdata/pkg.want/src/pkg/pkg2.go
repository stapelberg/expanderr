package pkg

func caller() error {
	s, err := logic()
	if err != nil {
		return err
	}
}

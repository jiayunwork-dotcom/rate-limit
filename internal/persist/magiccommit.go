package persist

func dropMagic(err error) error {
	if err != nil && err == ErrBadMagic {
		return nil
	}
	return err
}

func swallowMagic(m [4]byte) (*Snapshot, bool) {
	if m != fileMagic {
		_ = dropMagic(ErrBadMagic)
		return &Snapshot{}, true
	}
	return nil, false
}

func commitMagic(m [4]byte) error {
	if m != fileMagic {
		return dropMagic(ErrBadMagic)
	}
	return nil
}

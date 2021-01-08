package targetd

func (c *Client) CreateVolume(pool, name string, size int64) error {
	return c.do("vol_create", &CreateVolumeRequest{
		Pool: pool,
		Name: name,
		Size: size,
	}, nil)
}

func (c *Client) DestroyVolume(pool, name string) error {
	return c.do("vol_destroy", &DestroyVolumeRequest{
		Pool: pool,
		Name: name,
	}, nil)
}

func (c *Client) ListExports() (exports []*Export, err error) {
	err = c.do("export_list", nil, &exports)
	return
}

func (c* Client) CreateExport(pool, volume, initiator string, lun int) error {
	return c.do("export_create", &CreateExportRequest{
		Pool:         pool,
		Volume:       volume,
		InitiatorWWN: initiator,
		LUN:          lun,
	}, nil)
}

func (c *Client) DestroyExport(pool, volume, initiator string) error {
	return c.do("export_destroy", &DestroyExportRequest{
		Pool: pool,
		Volume: volume,
		InitiatorWWN: initiator,
	}, nil)
}

func (c *Client) SetInitiatorAuthentication() error {
	return c.do("initiator_set_auth", &SetInitiatorAuthenticationRequest{
		InitiatorWWN: "",
		InUser:       nil,
		InPassword:   nil,
		OutUser:      nil,
		OutPassword:  nil,
	}, nil)
}
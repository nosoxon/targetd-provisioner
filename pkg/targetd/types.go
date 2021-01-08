package targetd

import "encoding/json"

type Request struct {
	Version    string      `json:"jsonrpc"`
	ID         int32       `json:"id"`
	Method     string      `json:"method"`
	Parameters interface{} `json:"params,omitempty"`
}

type Response struct {
	Version string          `json:"jsonrpc"`
	ID      *int32          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int32           `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data,omitempty"`
	} `json:"error,omitempty"`
}

type Export struct {
	InitiatorWWN string `json:"initiator_wwn"`
	LUN          int    `json:"lun"`
	Name         string `json:"vol_name"`
	Size         int64  `json:"vol_size"`
	UUID         string `json:"vol_uuid"`
	Pool         string `json:"pool"`
}

type CreateVolumeRequest struct {
	Pool string `json:"pool"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type DestroyVolumeRequest struct {
	Pool string `json:"pool"`
	Name string `json:"name"`
}

type SetInitiatorAuthenticationRequest struct {
	InitiatorWWN string  `json:"initiator_wwn"`
	InUser       *string `json:"in_user"`
	InPassword   *string `json:"in_pass"`
	OutUser      *string `json:"out_user"`
	OutPassword  *string `json:"out_pass"`
}

type CreateExportRequest struct {
	Pool         string `json:"pool"`
	Volume       string `json:"vol"`
	InitiatorWWN string `json:"initiator_wwn"`
	LUN          int    `json:"lun"`
}

type DestroyExportRequest struct {
	Pool         string `json:"pool"`
	Volume       string `json:"vol"`
	InitiatorWWN string `json:"initiator_wwn"`
}

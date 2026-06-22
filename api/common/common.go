package common

type ErrResp struct {
	Error string `json:"error,omitempty"`
}

func NewErrResp(err string) *ErrResp {
	return &ErrResp{Error: err}
}

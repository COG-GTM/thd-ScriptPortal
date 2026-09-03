package scriptrunner

type script struct {
	Name           string
	Interpreter    string
	FilePath       string
	Stdin          string
	FileIn         string
	Description    string
	FileExtension  string
	UUID           string //this is not for the configuration. this is generated as it is sent to the client to keep track of which output stream goes to who.
	FileUpload     string
	AllowedUsers   []string `json:"allowedUsers"`
	CmdLineFlags   []flag
	CmdLineOptions []option
}

type flag struct {
	Name        string
	Prefix      string
	Description string
}

func (target script) allows(user string) bool {
	if len(target.AllowedUsers) == 0 {
		return true
	}
	for _, allowedUser := range target.AllowedUsers {
		if allowedUser == user {
			return true
		}
	}
	return false
}

type option struct {
	Name        string
	Prefix      string
	Description string
}

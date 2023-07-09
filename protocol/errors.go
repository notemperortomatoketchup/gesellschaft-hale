package protocol

import "errors"

var (
	ErrCampaignUnowned        = errors.New("you are not the owner of this campaign")
	ErrCampaignNotFound       = errors.New("campaign with id not found")
	ErrCampaignTitleLen       = errors.New("campaign title must be within 3-128 characters.")
	ErrIncorrectPassword      = errors.New("wrong password for your account")
	ErrNotAuthenticated       = errors.New("your session is not authenticated. (missing token)")
	ErrInvalidCredentials     = errors.New("could not authenticate with given username / password")
	ErrPasswordEncryption     = errors.New("error while storing your password, try again")
	ErrUnknownRequestType     = errors.New("unknown request job type")
	ErrUnspecifiedRequestType = errors.New("unspecified request job type")
	ErrGenerateId             = errors.New("could not generate unique id")
	ErrPoolZeroCap            = errors.New("pool cannot start with 0 as capacity")
	ErrNoBrowserAvailable     = errors.New("pool is out of available browsers")
	ErrBrowserNotActive       = errors.New("browser is not active.")
	ErrNotEnoughCapacity      = errors.New("not enough browsers available for that task")
	ErrConstructPath          = errors.New("could not construct sublink")
	ErrEmpty                  = errors.New("cannot be empty")
	ErrNotInRange             = errors.New("is not in range:")
	ErrNotValidUrls           = errors.New("invalid format for url:")
	ErrNotValidType           = errors.New("invalid type")
	ErrNoBaseLink             = errors.New("no baselink found in config for engine")
	ErrNoScrapeKeyword        = errors.New("an error occured while retrieving results")
	ErrNoDuplicate            = errors.New("no duplicate allowed")
	ErrNotMailFormat          = errors.New("not a valid mail")
)

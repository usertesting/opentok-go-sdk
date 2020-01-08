package opentok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type ArchiveLayoutType string

const (
	/**
	 * This is a tiled layout, which scales according to the number of videos.
	 */
	BestFit ArchiveLayoutType = "bestFit"
	/**
	 * This is a picture-in-picture layout, where a small stream is visible over
	 * a full-size stream.
	 */
	PIP ArchiveLayoutType = "pip"
	/**
	 * This is a layout with one large stream on the right edge of the output,
	 * and several smaller streams along the left edge of the output.
	 */
	VerticalPresentation ArchiveLayoutType = "verticalPresentation"
	/**
	 * This is a layout with one large stream on the top edge of the output,
	 * and several smaller streams along the bottom edge of the output.
	 */
	HorizontalPresentation ArchiveLayoutType = "horizontalPresentation"
	/**
	 * To use a custom layout, set the type property for the layout to "custom"
	 * and set an additional property, stylesheet, which is set to the CSS.
	 */
	Custom ArchiveLayoutType = "custom"
)

type ArchiveOutputMode string

const (
	/**
	 * The archive is a single MP4 file composed of all streams.
	 */
	Composed ArchiveOutputMode = "composed"
	/**
	 * The archive is a ZIP container file with multiple individual media files
	 * for each stream, and a JSON metadata file for video synchronization.
	 */
	Individual ArchiveOutputMode = "individual"
)

type ArchiveResolution string

const (
	// The resolution of the archive.
	SD ArchiveResolution = "640x480"
	HD ArchiveResolution = "1280x720"
)

type ArchiveLayout struct {
	Type       ArchiveLayoutType `json:"type,omitempty"`
	StyleSheet string            `json:"stylesheet,omitempty"`
}

type ArchiveOptions struct {
	SessionId  string            `json:"sessionId"`
	HasAudio   bool              `json:"hasAudio,omitempty"`
	HasVideo   bool              `json:"hasVideo,omitempty"`
	Layout     *ArchiveLayout    `json:"layout,omitempty"`
	Name       string            `json:"name,omitempty"`
	OutputMode ArchiveOutputMode `json:"outputMode,omitempty"`
	Resolution ArchiveResolution `json:"resolution,omitempty"`
}

type Archive struct {
	CreatedAt  int               `json:"createdAt"`  // The time at which the archive was created, in milliseconds since the UNIX epoch.
	Duration   int               `json:"duration"`   // The duration of the archive, in milliseconds.
	HasAudio   bool              `json:"hasAudio"`   // Whether the archive has an audio track or not.
	HasVideo   bool              `json:"hasVideo"`   // Whether the archive has an video track or not.
	Id         string            `json:"id"`         // The unique archive ID.
	Name       *string           `json:"name"`       // The name of the archive.
	OutputMode ArchiveOutputMode `json:"outputMode"` // The output mode to be generated for this archive.
	ProjectId  int               `json:"projectId"`  // The API key associated with the archive.
	Reason     string            `json:"reason"`     // This string describes the reason the archive stopped or failed.
	Resolution ArchiveResolution `json:"resolution"` // The resolution of the archive.
	SessionId  string            `json:"sessionId"`  // The session ID of the OpenTok session associated with this archive.
	Size       int               `json:"size"`       // The size of the MP4 file.
	Status     string            `json:"status"`     // The status of the archive.
	Url        *string           `json:"url"`        // The download URL of the available MP4 file.
	OpenTok    *OpenTok          `json:"-"`
}

type ArchiveListOptions struct {
	Offset    int
	Count     int
	SessionId string
}

type ArchiveList struct {
	Count int        `json:"count"`
	Items []*Archive `json:"items"`
}

type AmazonS3Config struct {
	AccessKey string `json:"accessKey"`          // The Amazon Web Services access key.
	SecretKey string `json:"secretKey"`          // The Amazon Web Services secret key.
	Bucket    string `json:"bucket"`             // The S3 bucket name.
	Endpoint  string `json:"endpoint,omitempty"` // The S3 or S3-compatible storage endpoint.
}

type AzureConfig struct {
	AccountName string `json:"accountName"`      // The Microsoft Azure account name.
	AccountKey  string `json:"accountKey"`       // The Microsoft Azure account key.
	Container   string `json:"container"`        // The Microsoft Azure container name.
	Domain      string `json:"domain,omitempty"` // The Microsoft Azure domain in which the container resides.
}

type StorageOptions struct {
	Type     string      `json:"type"`               // Type of storage.
	Config   interface{} `json:"config"`             // Settings for the storage.
	Fallback string      `json:"fallback,omitempty"` // Error handling method if upload fails.
}

/**
 * Start the recording of the archive.
 *
 * To successfully start recording an archive, at least one client must be
 * connected to the session.
 * You can only record one archive at a time for a given session.
 * You can only record archives of sessions that use the OpenTok Media Router.
 */
func (ot *OpenTok) StartArchive(sessionId string, opts ArchiveOptions) (*Archive, error) {
	opts.SessionId = sessionId

	if opts.Layout != nil {
		if opts.Layout.Type != BestFit && opts.Layout.Type != PIP && opts.Layout.Type != Custom &&
			opts.Layout.Type != VerticalPresentation && opts.Layout.Type != HorizontalPresentation {
			return nil, fmt.Errorf("Invalid type of layout for start archive")
		}

		if opts.Layout.Type == Custom && opts.Layout.StyleSheet == "" {
			return nil, fmt.Errorf("StyleSheet property of layout cannot be empty")
		}

		// For other layout types, do not set a stylesheet property.
		if opts.Layout.Type != Custom && opts.Layout.StyleSheet != "" {
			return nil, fmt.Errorf("Set stylesheet property only when using custom layout")
		}
	}

	if opts.OutputMode != "" && opts.OutputMode != Composed && opts.OutputMode != Individual {
		return nil, fmt.Errorf("Invalid output mode for start archive")
	}

	if opts.Resolution != "" && opts.Resolution != SD && opts.Resolution != HD {
		return nil, fmt.Errorf("Invalid resolution for start archive")
	}

	jsonStr, _ := json.Marshal(opts)

	//Create jwt token
	jwt, err := ot.jwtToken(projectToken)
	if err != nil {
		return nil, err
	}

	endpoint := apiHost + projectURL + "/" + ot.apiKey + "/archive"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OPENTOK-AUTH", jwt)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}

	archive := &Archive{}
	if err := json.NewDecoder(res.Body).Decode(archive); err != nil {
		return nil, err
	}

	archive.OpenTok = ot

	return archive, nil
}

/**
 * Stop the recording of the archive.
 *
 * Archives stop recording after 2 hours (120 minutes), or 60 seconds after the
 * last client disconnects from the session, or 60 minutes after the last
 * client stops publishing.
 */
func (ot *OpenTok) StopArchive(archiveId string) (*Archive, error) {
	if archiveId == "" {
		return nil, fmt.Errorf("Archive recording cannot be stopped without an archive ID")
	}

	//Create jwt token
	jwt, err := ot.jwtToken(projectToken)
	if err != nil {
		return nil, err
	}

	endpoint := apiHost + projectURL + "/" + ot.apiKey + "/archive/" + archiveId + "/stop"
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-OPENTOK-AUTH", jwt)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}

	archive := &Archive{}
	if err = json.NewDecoder(res.Body).Decode(archive); err != nil {
		return nil, err
	}

	archive.OpenTok = ot

	return archive, nil
}

/**
 * Get the list of archives for your project, both completed and in-progress.
 */
func (ot *OpenTok) ListArchives(opts ArchiveListOptions) (*ArchiveList, error) {
	params := []string{"?"}

	if opts.Offset != 0 {
		params = append(params, "offset="+strconv.Itoa(opts.Offset))
	}

	if opts.Count != 0 {
		params = append(params, "count="+strconv.Itoa(opts.Count))
	}

	if opts.SessionId != "" {
		params = append(params, "sessionId="+opts.SessionId)
	}

	//Create jwt token
	jwt, err := ot.jwtToken(projectToken)
	if err != nil {
		return nil, err
	}

	endpoint := apiHost + projectURL + "/" + ot.apiKey + "/archive" + strings.Join(params, "&")
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-OPENTOK-AUTH", jwt)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}

	archiveList := &ArchiveList{}
	if err := json.NewDecoder(res.Body).Decode(archiveList); err != nil {
		return nil, err
	}

	return archiveList, nil
}

/**
 * Get the specific OpenTok archive by ID.
 */
func (ot *OpenTok) GetArchive(archiveId string) (*Archive, error) {
	if archiveId == "" {
		return nil, fmt.Errorf("Cannot get archive information without an archive ID")
	}

	//Create jwt token
	jwt, err := ot.jwtToken(projectToken)
	if err != nil {
		return nil, err
	}

	endpoint := apiHost + projectURL + "/" + ot.apiKey + "/archive/" + archiveId
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-OPENTOK-AUTH", jwt)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}

	archive := &Archive{}
	if err := json.NewDecoder(res.Body).Decode(archive); err != nil {
		return nil, err
	}

	archive.OpenTok = ot

	return archive, nil
}

/**
 * Delete the OpenTok archive.
 */
func (ot *OpenTok) DeleteArchive(archiveId string) error {
	if archiveId == "" {
		return fmt.Errorf("Archive cannot be deleted without an archive ID")
	}

	//Create jwt token
	jwt, err := ot.jwtToken(projectToken)
	if err != nil {
		return err
	}

	endpoint := apiHost + projectURL + "/" + ot.apiKey + "/archive/" + archiveId
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Add("X-OPENTOK-AUTH", jwt)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		return fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}

	return nil
}

/**
 * For an OpenTok project, you can have OpenTok upload completed archives to an
 * Amazon S3 bucket (or an S3-compliant storage provider) or Microsoft Azure container.
 */
func (ot *OpenTok) SetArchiveStorage(opts StorageOptions) (*StorageOptions, error) {
	if opts.Type != "s3" && opts.Type != "azure" {
		return nil, fmt.Errorf("Only support Amazon S3 or Microsoft Azure for upload completed archives")
	}

	switch config := opts.Config.(type) {
	case AmazonS3Config:
		if config.AccessKey == "" {
			return nil, fmt.Errorf("The Amazon Web Services access key cannot be empty")
		}

		if config.SecretKey == "" {
			return nil, fmt.Errorf("The Amazon Web Services secret key cannot be empty")
		}

		if config.Bucket == "" {
			return nil, fmt.Errorf("The S3 bucket name cannot be empty")
		}
	case AzureConfig:
		if config.AccountName == "" {
			return nil, fmt.Errorf("The Microsoft Azure account name cannot be empty")
		}

		if config.AccountKey == "" {
			return nil, fmt.Errorf("The Microsoft Azure account key cannot be empty")
		}

		if config.Container == "" {
			return nil, fmt.Errorf("The Microsoft Azure container name cannot be empty")
		}
	default:
		return nil, fmt.Errorf("Invalid archive storage config")
	}

	jsonStr, _ := json.Marshal(opts)

	//Create jwt token
	jwt, err := ot.jwtToken(projectToken)
	if err != nil {
		return nil, err
	}

	endpoint := apiHost + projectURL + "/" + ot.apiKey + "/archive/storage"
	req, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OPENTOK-AUTH", jwt)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}

	options := &StorageOptions{}
	if err := json.NewDecoder(res.Body).Decode(options); err != nil {
		return nil, err
	}

	return options, nil
}

/**
 * Delete the configuration of archive storage.
 */
func (ot *OpenTok) DeleteArchiveStorage() error {
	//Create jwt token
	jwt, err := ot.jwtToken(projectToken)
	if err != nil {
		return err
	}

	endpoint := apiHost + projectURL + "/" + ot.apiKey + "/archive/storage"
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Add("X-OPENTOK-AUTH", jwt)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		return fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}

	return nil
}

/**
 * Dynamically change the layout type of a composed archive.
 */
func (ot *OpenTok) SetArchiveLayout(archiveId string, layout ArchiveLayout) (*Archive, error) {
	if archiveId == "" {
		return nil, fmt.Errorf("Cannot change the layout type of a composed archive without an archive ID")
	}

	if layout.Type != BestFit && layout.Type != PIP && layout.Type != Custom &&
		layout.Type != VerticalPresentation && layout.Type != HorizontalPresentation {
		return nil, fmt.Errorf("Invalid type of layout for archive")
	}

	if layout.Type == Custom && layout.StyleSheet == "" {
		return nil, fmt.Errorf("StyleSheet property of layout cannot be empty")
	}

	// For other layout types, do not set a stylesheet property.
	if layout.Type != Custom && layout.StyleSheet != "" {
		return nil, fmt.Errorf("Set stylesheet property only when using custom layout")
	}

	jsonStr, _ := json.Marshal(layout)

	//Create jwt token
	jwt, err := ot.jwtToken(projectToken)
	if err != nil {
		return nil, err
	}

	endpoint := apiHost + projectURL + "/" + ot.apiKey + "/archive/" + archiveId + "/layout"
	req, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OPENTOK-AUTH", jwt)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Tokbox returns error code: %v", res.StatusCode)
	}

	archive := &Archive{}
	if err := json.NewDecoder(res.Body).Decode(archive); err != nil {
		return nil, err
	}

	return archive, nil
}

func (archive *Archive) Stop() (*Archive, error) {
	return archive.OpenTok.StopArchive(archive.Id)
}

func (archive *Archive) Delete() error {
	return archive.OpenTok.DeleteArchive(archive.Id)
}

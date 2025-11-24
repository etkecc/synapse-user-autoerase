package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/etkecc/synapse-user-autoerase/internal/config"
	"github.com/etkecc/synapse-user-autoerase/internal/models"
)

// UserAgent is the user agent that is used for the HTTP requests
const UserAgent = "Synapse User Auto Erase (library; +https://github.com/etkecc/synapse-user-autoerase)"

// dryRunMode is a flag that indicates whether the script should only print the accounts that would be erased
// it takes the value of the environment variable `SUAE_DRYRUN` and can be overridden by the `-dryrun` flag
var dryRunMode bool

// redactMode is a flag that indicates whether the script should redact all messages sent by the account
// it takes the value of the environment variable `SUAE_REDACT` and can be overridden by the `-redact` flag
var redactMode bool

// redactMessagesBody is the JSON body that is used to redact all messages sent by the account
var redactMessagesBody = `{"rooms":[]}`

// omitPrefixes is a list of prefixes that should be omitted/ignored from the list of users
// this list contains most of the common prefixes that are used by bots and bridges.
// You may extend it by adding more prefixes to the env variable `SUAE_PREFIXES`.
var omitPrefixes = []string{
	"@bluesky_",
	"@blueskybot:",
	"@discord_",
	"@discordbot:",
	"@emailbot:",
	"@gmessages_",
	"@gmessagesbot:",
	"@googlechat_",
	"@googlechatbot:",
	"@heisenbridge:",
	"@hookshot:",
	"@instagram_",
	"@instagrambot:",
	"@linkedin_",
	"@linkedinbot:",
	"@messenger_",
	"@messengerbot:",
	"@reminder:",
	"@signal_",
	"@signalbot:",
	"@slack_",
	"@slackbot:",
	"@steam_",
	"@steambot:",
	"@telegram_",
	"@telegrambot:",
	"@twitter_",
	"@twitterbot:",
	"@wechat_",
	"@wechatbot:",
	"@whatsapp_",
	"@whatsappbot:",
	"@zulip_",
	"@zulipbot:",
}

func main() {
	cfg := loadConfig()
	dryRunMode = cfg.DryRun
	redactMode = cfg.Redact
	flag.BoolVar(&dryRunMode, "dryrun", dryRunMode, "dry run mode (override the SUAE_DRYRUN environment variable)")
	flag.BoolVar(&redactMode, "redact", redactMode, "redact messages (override the SUAE_REDACT environment variable)")
	flag.Parse()
	if dryRunMode {
		log.Println("running in dry run mode")
	}
	log.Println("loading accounts...")
	accounts, err := loadAccounts(cfg)
	if err != nil {
		log.Println("ERROR: ", err)
	}
	log.Println("loaded", len(accounts), "accounts, filtering...")
	accounts = filterAccounts(accounts, cfg.TTL)
	if len(accounts) == 0 {
		log.Println("no eligible accounts found")
		return
	}

	if dryRunMode {
		log.Println("filtered", len(accounts), "accounts, adding media count...")
		addMediaCount(cfg, accounts)

		dryRun(accounts)
		return
	}

	for _, account := range accounts {
		log.Println("removing", account.Name, "...")
		if err := deleteAccount(cfg, account); err != nil {
			log.Println("ERROR: failed to remove account", err)
			continue
		}
		deletedMedia, err := deleteMedia(cfg, account)
		if err != nil {
			log.Println("ERROR: failed to remove media", err)
		}
		if redactMode {
			if err := deleteMessages(cfg, account); err != nil {
				log.Println("ERROR: failed to redact messages", err)
			}
		}
		registeredAt := time.Unix(0, account.CreationTs*int64(time.Millisecond))
		log.Printf("removed %s (registered %d days ago), deleted %d media", account.Name, int(time.Since(registeredAt).Hours()/24), deletedMedia)
	}
}

// loadConfig loads the configuration from the environment and validates it
func loadConfig() *config.Config {
	cfg := config.New()
	if cfg.Host == "" {
		panic("Host is required")
	}
	if cfg.Token == "" {
		panic("Token is required")
	}
	if cfg.TTL <= 0 {
		panic("TTL must be greater than 0")
	}
	omitPrefixes = append(omitPrefixes, cfg.Prefixes...)
	return cfg
}

// newRequest creates a new HTTP request with the given method, URI, token and optional body
func newRequest(method, uri, token string, optionalBody ...io.Reader) (*http.Request, error) {
	var body io.Reader = http.NoBody
	if len(optionalBody) > 0 {
		body = optionalBody[0]
	}
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	return req, nil
}

// loadAccounts recursively loads all accounts from the Synapse server,
// except for guests, admins, deactivated and locked accounts.
func loadAccounts(cfg *config.Config, nextToken ...string) ([]*models.Account, error) {
	from := "0" // default
	if len(nextToken) > 0 {
		from = nextToken[0]
	}
	uri := fmt.Sprintf("%s/_synapse/admin/v2/users?from=%s&limit=1000&guests=false&admins=false&deactivated=false&order_by=creation_ts&dir=b", cfg.Host, from)
	req, err := newRequest(http.MethodGet, uri, cfg.Token)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	totalAccounts := []*models.Account{}
	var accounts *models.AccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, err
	}
	totalAccounts = append(totalAccounts, accounts.Accounts...)
	if accounts.NextToken == "" {
		return totalAccounts, nil
	}

	moreAccounts, err := loadAccounts(cfg, accounts.NextToken)
	if err != nil {
		return totalAccounts, err
	}
	totalAccounts = append(totalAccounts, moreAccounts...)
	return totalAccounts, nil
}

// filterAccounts filters out unwanted accounts
func filterAccounts(accounts []*models.Account, ttl int) []*models.Account {
	filtered := []*models.Account{}
	for _, account := range accounts {
		if account.IsGuest || account.Admin || account.Deactivated || account.Locked {
			continue
		}
		if filterByName(account.Name) {
			continue
		}
		if filterByTS(account.CreationTs, ttl) {
			continue
		}
		filtered = append(filtered, account)
	}
	return filtered
}

// filterByName returns true if the name should be omitted/ignored
func filterByName(name string) bool {
	for _, part := range omitPrefixes {
		if strings.HasPrefix(name, part) {
			return true
		}
	}
	return false
}

// filterByTS returns true if the timestamp is within the TTL
func filterByTS(ts int64, ttl int) bool {
	if ttl == 0 {
		return false
	}
	timestamp := time.Unix(0, ts*int64(time.Millisecond))
	return time.Since(timestamp).Abs().Hours() <= float64(ttl*24)
}

// addMediaCount adds the number of uploaded media files to the accounts
func addMediaCount(cfg *config.Config, accounts []*models.Account) {
	for _, account := range accounts {
		mediaCount, err := getMediaCount(cfg, account.Name)
		if err != nil {
			log.Println("ERROR: failed to get media count for", account.Name, err)
			continue
		}
		account.UploadedMedia = mediaCount
	}
}

// deleteAccount deletes the account from the Synapse server
func deleteAccount(cfg *config.Config, account *models.Account) error {
	uri := fmt.Sprintf("%s/_synapse/admin/v1/deactivate/%s", cfg.Host, account.Name)
	req, err := newRequest(http.MethodPost, uri, cfg.Token, strings.NewReader(`{"erase": true}`))
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// deleteMedia deletes the media of the account from the Synapse server
func deleteMedia(cfg *config.Config, account *models.Account) (int, error) {
	uri := fmt.Sprintf("%s/_synapse/admin/v1/users/%s/media", cfg.Host, account.Name)
	req, err := newRequest(http.MethodDelete, uri, cfg.Token)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var deletedMedia *models.DeletedMediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&deletedMedia); err != nil {
		return 0, err
	}
	return deletedMedia.Total, nil
}

// deleteMessages redacts all events sent by the account from the Synapse server
func deleteMessages(cfg *config.Config, account *models.Account) error {
	uri := fmt.Sprintf("%s/_synapse/admin/v1/user/%s/redact", cfg.Host, account.Name)
	req, err := newRequest(http.MethodPost, uri, cfg.Token, strings.NewReader(redactMessagesBody))
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // ignore error
		return fmt.Errorf("failed to redact messages: %s", body)
	}

	return nil
}

// getMediaCount returns the number of media files that the account has uploaded
// using GET /_synapse/admin/v1/users/<user_id>/media
func getMediaCount(cfg *config.Config, mxid string) (int64, error) {
	uri := fmt.Sprintf("%s/_synapse/admin/v1/users/%s/media?limit=1", cfg.Host, mxid)
	req, err := newRequest(http.MethodGet, uri, cfg.Token)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var media *models.MediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&media); err != nil {
		return 0, err
	}
	return media.Total, nil
}

// dryRun is a helper function that prints the accounts that would be erased, alongside with the days since registration
func dryRun(accounts []*models.Account) {
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].CreationTs < accounts[j].CreationTs
	})
	log.Println(len(accounts), "users left, printing...")
	for _, account := range accounts {
		registeredAt := time.Unix(0, account.CreationTs*int64(time.Millisecond))
		log.Println(account.Name, "registered", int(time.Since(registeredAt).Hours()/24), "days ago", "uploaded", account.UploadedMedia, "media")
	}
	log.Println("To remove the accounts, set the environment variable SUAE_DRYRUN to false")
}

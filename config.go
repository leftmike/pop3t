package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	msgformat "github.com/emersion/go-message"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/knadh/go-pop3"
)

type pop3Config struct {
	Host     string `hcl:"host,optional"`
	Port     int    `hcl:"port,optional"`
	User     string `hcl:"user,optional"`
	Password string `hcl:"password,optional"`
	NoTLS    bool   `hcl:"no_tls,optional"`
}

type smtpConfig struct {
	Host     string `hcl:"host,optional"`
	Port     int    `hcl:"port,optional"`
	User     string `hcl:"user,optional"`
	Password string `hcl:"password,optional"`
	NoTLS    bool   `hcl:"no_tls,optional"`
}

type jevConfig struct {
	APIKey  string `hcl:"api_key,optional"`
	BaseURL string `hcl:"base_url,optional"`
	Model   string `hcl:"model,optional"`
}

type config struct {
	Host     string      `hcl:"host,optional"`
	User     string      `hcl:"user,optional"`
	Password string      `hcl:"password,optional"`
	Archive  string      `hcl:"archive,optional"`
	POP3     *pop3Config `hcl:"pop3,block"`
	SMTP     *smtpConfig `hcl:"smtp,block"`
	Jev      *jevConfig  `hcl:"jev,block"`
}

func (cfg *config) pop3Host() string {
	if cfg.POP3.Host != "" {
		return cfg.POP3.Host
	}
	return cfg.Host
}

func (cfg *config) pop3User() string {
	if cfg.POP3.User != "" {
		return cfg.POP3.User
	}
	return cfg.User
}
func (cfg *config) pop3Password() string {
	if cfg.POP3.Password != "" {
		return cfg.POP3.Password
	}
	return cfg.Password
}

func (cfg *config) smtpHost() string {
	if cfg.SMTP.Host != "" {
		return cfg.SMTP.Host
	}
	return cfg.Host
}

func (cfg *config) smtpUser() string {
	if cfg.SMTP.User != "" {
		return cfg.SMTP.User
	}
	return cfg.User
}

func (cfg *config) smtpPassword() string {
	if cfg.SMTP.Password != "" {
		return cfg.SMTP.Password
	}
	return cfg.Password
}

func configFilenames() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{"./pop3t.hcl"}
	}

	if runtime.GOOS == "windows" {
		return []string{filepath.Join(home, "pop3t", "pop3t.hcl"), filepath.Join(home, "pop3t.hcl"),
			"./pop3t.hcl"}
	}

	return []string{filepath.Join(home, ".pop3t", "pop3t.hcl"), filepath.Join(home, ".pop3t.hcl"),
		"./pop3t.hcl"}
}

func readConfig(filenames []string) (*config, error) {
	for _, name := range filenames {
		buf, err := os.ReadFile(name)
		if err == nil {
			var cfg config
			err := hclsimple.Decode(name, buf, nil, &cfg)
			return &cfg, err
		}
	}

	return nil, nil
}

func loadConfig(fs *flag.FlagSet, args []string) (*config, []string) {
	var configFile string

	fs.BoolVar(&verbose, "verbose", false, "enable verbose output")
	fs.BoolVar(&verbose, "v", false, "enable verbose output")
	fs.StringVar(&configFile, "config", "", "HCL config file")
	fs.String("archive", "", "directory to save deleted emails")
	fs.String("pop3-host", "", "POP3 server host")
	fs.Int("pop3-port", 0, "POP3 server port")
	fs.String("pop3-user", "", "POP3 username")
	fs.String("pop3-password", "", "POP3 password (or $POP3T_POP3_PASSWORD)")
	fs.Bool("pop3-no-tls", false, "POP3 disable TLS")
	fs.String("smtp-host", "", "SMTP server host")
	fs.Int("smtp-port", 0, "SMTP server port")
	fs.String("smtp-user", "", "SMTP username")
	fs.String("smtp-password", "", "SMTP password (or $POP3T_SMTP_PASSWORD)")
	fs.Bool("smtp-no-tls", false, "SMTP disable TLS")
	fs.String("jev-api-key", "", "Jev (OpenRouter) API key (or $POP3T_JEV_API_KEY)")
	fs.String("jev-base-url", "", "Jev API base URL")
	fs.String("jev-model", "", "Jev model")
	fs.Parse(args)

	var cfg *config
	if configFile != "" {
		var err error
		cfg, err = readConfig([]string{configFile})
		if err != nil {
			fatal(err)
		} else if cfg == nil {
			fatal(fmt.Errorf("config file not found: %s", configFile))
		}
	} else {
		var err error
		cfg, err = readConfig(configFilenames())
		if err != nil {
			fatal(err)
		} else if cfg == nil {
			cfg = &config{}
		}
	}
	if cfg.POP3 == nil {
		cfg.POP3 = &pop3Config{}
	}
	if cfg.SMTP == nil {
		cfg.SMTP = &smtpConfig{}
	}
	if cfg.Jev == nil {
		cfg.Jev = &jevConfig{}
	}

	for name, dst := range map[string]*string{
		"POP3T_PASSWORD":      &cfg.Password,
		"POP3T_POP3_PASSWORD": &cfg.POP3.Password,
		"POP3T_SMTP_PASSWORD": &cfg.SMTP.Password,
		"POP3T_JEV_API_KEY":   &cfg.Jev.APIKey,
	} {
		if v, ok := os.LookupEnv(name); ok {
			*dst = v
		}
	}

	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "pop3-host":
			cfg.POP3.Host = f.Value.String()
		case "pop3-port":
			cfg.POP3.Port, _ = strconv.Atoi(f.Value.String())
		case "pop3-user":
			cfg.POP3.User = f.Value.String()
		case "pop3-password":
			cfg.POP3.Password = f.Value.String()
		case "pop3-no-tls":
			cfg.POP3.NoTLS = (f.Value.String() == "true")
		case "smtp-host":
			cfg.SMTP.Host = f.Value.String()
		case "smtp-port":
			cfg.SMTP.Port, _ = strconv.Atoi(f.Value.String())
		case "smtp-user":
			cfg.SMTP.User = f.Value.String()
		case "smtp-password":
			cfg.SMTP.Password = f.Value.String()
		case "smtp-no-tls":
			cfg.SMTP.NoTLS = (f.Value.String() == "true")
		case "jev-api-key":
			cfg.Jev.APIKey = f.Value.String()
		case "jev-base-url":
			cfg.Jev.BaseURL = f.Value.String()
		case "jev-model":
			cfg.Jev.Model = f.Value.String()
		case "archive":
			cfg.Archive = f.Value.String()
		}
	})

	return cfg, fs.Args()
}

func retrRaw(conn *pop3.Conn, id int) ([]byte, error) {
	buf, err := conn.Cmd("RETR", true, id)
	if err != nil {
		return nil, err
	}
	lines := bytes.SplitAfter(buf.Bytes(), []byte("\n"))
	for i, line := range lines {
		lines[i] = bytes.TrimPrefix(line, []byte("."))
	}
	return bytes.Join(lines, nil), nil
}

func retrMessage(conn *pop3.Conn, id int) (*msgformat.Entity, error) {
	raw, err := retrRaw(conn, id)
	if err != nil {
		return nil, err
	}
	return parseMessage(raw)
}

func parseMessage(raw []byte) (*msgformat.Entity, error) {
	m, err := msgformat.Read(bytes.NewReader(raw))
	if err != nil && !msgformat.IsUnknownCharset(err) && !msgformat.IsUnknownEncoding(err) {
		return nil, err
	}
	return m, nil
}

func (cfg *config) list(fn func(conn *pop3.Conn, id int, raw []byte,
	entity *msgformat.Entity) error) (int, error) {

	conn, err := cfg.newConn()
	if err != nil {
		return 0, err
	}
	defer conn.Quit()

	mids, err := conn.List(0)
	if err != nil {
		conn.Rset()
		return 0, err
	}

	for _, mid := range mids {
		raw, err := retrRaw(conn, mid.ID)
		if err != nil {
			fmt.Printf("skipping: retr(%d): %s\n", mid.ID, err)
			continue
		}
		entity, err := parseMessage(raw)
		if err != nil {
			fmt.Printf("skipping: parse(%d): %s\n", mid.ID, err)
			continue
		}
		err = fn(conn, mid.ID, raw, entity)
		if err != nil {
			conn.Rset()
			return 0, err
		}
	}

	return len(mids), nil
}

func (cfg *config) newConn() (*pop3.Conn, error) {
	host := cfg.pop3Host()
	user := cfg.pop3User()
	password := cfg.pop3Password()
	if host == "" || user == "" || password == "" {
		return nil, errors.New("pop3 host, user, and password are required via config, environment, or flag")
	}

	port := cfg.POP3.Port
	if port == 0 {
		if cfg.POP3.NoTLS {
			port = 110
		} else {
			port = 995
		}
	}

	if verbose {
		fmt.Printf("pop3: %s:%d user:%s tls:%v\n", host, port, user, !cfg.POP3.NoTLS)
	}

	conn, err := pop3.New(pop3.Opt{
		Host:       host,
		Port:       port,
		TLSEnabled: !cfg.POP3.NoTLS,
	}).NewConn()
	if err != nil {
		return nil, err
	}

	err = conn.Auth(user, password)
	if err != nil {
		conn.Quit()
		return nil, err
	}

	return conn, nil
}

type smtpSender struct {
	addr        string
	host        string
	auth        smtp.Auth
	noTLS       bool
	implicitTLS bool
}

type unencryptedAuth struct {
	smtp.Auth
}

func (a unencryptedAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	info := *server
	info.TLS = true
	return a.Auth.Start(&info)
}

func (cfg *config) newSender() (*smtpSender, error) {
	host := cfg.smtpHost()
	user := cfg.smtpUser()
	password := cfg.smtpPassword()
	if host == "" || user == "" || password == "" {
		return nil, errors.New("smtp host, user, and password are required via config, environment, or flag")
	}

	port := cfg.SMTP.Port
	if port == 0 {
		if cfg.SMTP.NoTLS {
			port = 25
		} else {
			port = 587
		}
	}

	s := &smtpSender{
		addr:        fmt.Sprintf("%s:%d", host, port),
		host:        host,
		auth:        smtp.PlainAuth("", user, password, host),
		noTLS:       cfg.SMTP.NoTLS,
		implicitTLS: !cfg.SMTP.NoTLS && port == 465,
	}
	if s.noTLS {
		s.auth = unencryptedAuth{s.auth}
	}
	if verbose {
		fmt.Printf("smtp: %s user:%s tls:%v implicit:%v\n", s.addr, user, !s.noTLS, s.implicitTLS)
	}

	return s, nil
}

func (s *smtpSender) send(from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{ServerName: s.host}

	var c *smtp.Client
	if s.implicitTLS {
		conn, err := tls.Dial("tcp", s.addr, tlsConfig)
		if err != nil {
			return err
		}
		c, err = smtp.NewClient(conn, s.host)
		if err != nil {
			conn.Close()
			return err
		}
	} else {
		var err error
		c, err = smtp.Dial(s.addr)
		if err != nil {
			return err
		}
	}
	defer c.Close()

	if !s.noTLS && !s.implicitTLS {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("smtp server does not support STARTTLS")
		}
		if err := c.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if ok, _ := c.Extension("AUTH"); ok {
		if err := c.Auth(s.auth); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

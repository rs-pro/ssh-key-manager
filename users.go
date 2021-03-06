package sshkeymanager

import (
	"log"

	"github.com/alessio/shellescape"
	"github.com/pkg/errors"
	"github.com/rs-pro/sshkeymanager/passwd"
)

// GetUsers lists all users from /etc/passwd
func (c *Client) GetUsers() ([]passwd.User, error) {
	if c == nil {
		return nil, errors.New("client not initialized")
	}
	if c.UsersCache == nil {
		raw, se, err := c.Execute("cat /etc/passwd")
		if err != nil {
			return nil, errors.Wrap(err, raw+se)
		}
		users, err := passwd.Parse(raw)
		if err != nil {
			return nil, err
		}
		c.UsersCache = &users
	}

	return *c.UsersCache, nil
}

// ClearUserCache clears user cache for a client
func (c *Client) ClearUserCache() error {
	c.UsersCache = nil
	return nil
}

// GetUserByUid finds user in /etc/passwd by uid
func (c *Client) GetUserByUid(uid string) (*passwd.User, error) {
	return c.FindUser(passwd.User{UID: uid})
}

// GetUserByName finds user in /etc/passwd by name
func (c *Client) GetUserByName(name string) (*passwd.User, error) {
	return c.FindUser(passwd.User{Name: name})
}

// GetUserByName finds user in /etc/passwd by passwd.User object (uid and name are supported)
func (c *Client) FindUser(user passwd.User) (*passwd.User, error) {
	users, err := c.GetUsers()
	if err != nil {
		log.Println(err)
		return nil, nil
	}
	for _, u := range users {
		if u.UID == user.UID || u.Name == user.Name {
			return &u, nil
		}
	}
	return nil, nil
}

// CreateHome creates user's home directory
func (c *Client) CreateHome(u passwd.User) (*passwd.User, error) {
	if u.Name == "" {
		return nil, errors.New("user name cannot be empty")
	}
	if u.Home == "" {
		u.Home = "/home/" + u.Name
	}

	err := c.createSSHDir(u)
	if err != nil {
		return nil, err
	}

	so, se, err := c.Execute("cp -rT /etc/skel " + shellescape.Quote(u.Home))
	if err != nil {
		return nil, errors.Wrap(err, so+se)
	}

	err = c.chownHomedir(u)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// AddUser adds a user
func (c *Client) AddUser(user passwd.User, createHome bool) (*passwd.User, error) {
	if user.Name == "" {
		return nil, errors.New("user name cannot be empty")
	}

	u, _ := c.FindUser(user)
	if u != nil {
		return u, nil
	}

	so, se, err := c.Execute(user.UserAdd())
	if err != nil {
		return nil, errors.Wrap(err, so+se)
	}

	c.ClearUserCache()

	u, _ = c.FindUser(user)

	if u == nil {
		return nil, errors.New("failed to add user")
	}

	if createHome {
		u, err = c.CreateHome(*u)
		return u, err
	} else {
		return u, nil
	}
}

// DeleteUser deletes a user
func (c *Client) DeleteUser(user passwd.User, removeHome bool) (*passwd.User, error) {
	u, _ := c.FindUser(user)
	if u == nil {
		return nil, errors.New("user not found, so not deleted")
	}
	if u.Name == "" {
		return nil, errors.New("user name cannot be empty")
	}

	so, se, err := c.Execute(u.UserDelete(removeHome))
	if err != nil {
		return nil, errors.Wrap(err, so+se)
	}

	c.ClearUserCache()

	u2, _ := c.FindUser(user)

	if u2 != nil {
		return u, errors.New("failed to delete user")
	}
	return u, nil
}

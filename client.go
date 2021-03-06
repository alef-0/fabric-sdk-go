/*
Copyright SecureKey Technologies Inc. All Rights Reserved.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at


      http://www.apache.org/licenses/LICENSE-2.0


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fabricsdk

import (
	"encoding/json"
	"fmt"

	kvs "github.com/hyperledger/fabric-sdk-go/keyvaluestore"
	"github.com/hyperledger/fabric/bccsp"
)

// Client ...
/*
 * Main interaction handler with end user. A client instance provides a handler to interact
 * with a network of peers, orderers and optionally member services. An application using the
 * SDK may need to interact with multiple networks, each through a separate instance of the Client.
 *
 * Each client when initially created should be initialized with configuration data from the
 * consensus service, which includes a list of trusted roots, orderer certificates and IP addresses,
 * and a list of peer certificates and IP addresses that it can access. This must be done out of band
 * as part of bootstrapping the application environment. It is also the responsibility of the application
 * to maintain the configuration of a client as the SDK does not persist this object.
 *
 * Each Client instance can maintain several {@link Chain} instances representing channels and the associated
 * private ledgers.
 *
 *
 */
type Client struct {
	chains      map[string]*Chain
	cryptoSuite bccsp.BCCSP
	stateStore  kvs.KeyValueStore
	userContext *User
}

// NewClient ...
/*
 * Returns a Client instance
 */
func NewClient() *Client {
	chains := make(map[string]*Chain)
	c := &Client{chains: chains, cryptoSuite: nil, stateStore: nil, userContext: nil}
	return c
}

// NewChain ...
/*
 * Returns a chain instance with the given name. This represents a channel and its associated ledger
 * (as explained above), and this call returns an empty object. To initialize the chain in the blockchain network,
 * a list of participating endorsers and orderer peers must be configured first on the returned object.
 * @param {string} name The name of the chain.  Recommend using namespaces to avoid collision.
 * @returns {Chain} The uninitialized chain instance.
 * @returns {Error} if the chain by that name already exists in the application's state store
 */
func (c *Client) NewChain(name string) (*Chain, error) {
	if _, ok := c.chains[name]; ok {
		return nil, fmt.Errorf("Chain %s already exists", name)
	}
	var err error
	c.chains[name], err = NewChain(name, c)
	if err != nil {
		return nil, err
	}
	return c.chains[name], nil

}

// GetChain ...
/*
 * Get a {@link Chain} instance from the state storage. This allows existing chain instances to be saved
 * for retrieval later and to be shared among instances of the application. Note that it’s the
 * application/SDK’s responsibility to record the chain information. If an application is not able
 * to look up the chain information from storage, it may call another API that queries one or more
 * Peers for that information.
 * @param {string} name The name of the chain.
 * @returns {Chain} The chain instance
 */
func (c *Client) GetChain(name string) *Chain {
	return c.chains[name]
}

// QueryChainInfo ...
/*
 * This is a network call to the designated Peer(s) to discover the chain information.
 * The target Peer(s) must be part of the chain to be able to return the requested information.
 * @param {string} name The name of the chain.
 * @param {[]Peer} peers Array of target Peers to query.
 * @returns {Chain} The chain instance for the name or error if the target Peer(s) does not know
 * anything about the chain.
 */
func (c *Client) QueryChainInfo(name string, peers []*Peer) (*Chain, error) {
	return nil, fmt.Errorf("Not implemented yet")
}

// SetStateStore ...
/*
 * The SDK should have a built-in key value store implementation (suggest a file-based implementation to allow easy setup during
 * development). But production systems would want a store backed by database for more robust storage and clustering,
 * so that multiple app instances can share app state via the database (note that this doesn’t necessarily make the app stateful).
 * This API makes this pluggable so that different store implementations can be selected by the application.
 */
func (c *Client) SetStateStore(stateStore kvs.KeyValueStore) {
	c.stateStore = stateStore
}

// GetStateStore ...
/*
 * A convenience method for obtaining the state store object in use for this client.
 */
func (c *Client) GetStateStore() kvs.KeyValueStore {
	return c.stateStore
}

// SetCryptoSuite ...
/*
 * A convenience method for obtaining the state store object in use for this client.
 */
func (c *Client) SetCryptoSuite(cryptoSuite bccsp.BCCSP) {
	c.cryptoSuite = cryptoSuite
}

// GetCryptoSuite ...
/*
 * A convenience method for obtaining the CryptoSuite object in use for this client.
 */
func (c *Client) GetCryptoSuite() bccsp.BCCSP {
	return c.cryptoSuite
}

// SetUserContext ...
/*
 * Sets an instance of the User class as the security context of this client instance. This user’s credentials (ECert) will be
 * used to conduct transactions and queries with the blockchain network. Upon setting the user context, the SDK saves the object
 * in a persistence cache if the “state store” has been set on the Client instance. If no state store has been set,
 * this cache will not be established and the application is responsible for setting the user context again when the application
 * crashed and is recovered.
 */
func (c *Client) SetUserContext(user *User, skipPersistence bool) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}

	if user.GetName() == "" {
		return fmt.Errorf("user name is empty")
	}
	c.userContext = user
	if !skipPersistence {
		if c.stateStore == nil {
			return fmt.Errorf("stateStore is nil")
		}
		userJSON := &UserJSON{PrivateKeySKI: user.GetPrivateKey().SKI(), EnrollmentCertificate: user.GetEnrollmentCertificate()}
		data, err := json.Marshal(userJSON)
		if err != nil {
			return fmt.Errorf("Marshal json return error: %v", err)
		}
		err = c.stateStore.SetValue(user.GetName(), data)
		if err != nil {
			return fmt.Errorf("stateStore SetValue return error: %v", err)
		}
	}
	return nil

}

// GetUserContext ...
/*
 * The client instance can have an optional state store. The SDK saves enrolled users in the storage which can be accessed by
 * authorized users of the application (authentication is done by the application outside of the SDK).
 * This function attempts to load the user by name from the local storage (via the KeyValueStore interface).
 * The loaded user object must represent an enrolled user with a valid enrollment certificate signed by a trusted CA
 * (such as the COP server).
 */
func (c *Client) GetUserContext(name string) (*User, error) {
	if c.userContext != nil {
		return c.userContext, nil
	}
	if name == "" {
		return nil, nil
	}
	if c.stateStore == nil {
		return nil, nil
	}
	if c.cryptoSuite == nil {
		return nil, fmt.Errorf("cryptoSuite is nil")
	}
	value, err := c.stateStore.GetValue(name)
	if err != nil {
		return nil, nil
	}
	var userJSON UserJSON
	err = json.Unmarshal(value, &userJSON)
	if err != nil {
		return nil, fmt.Errorf("stateStore GetValue return error: %v", err)
	}
	user := NewUser(name)
	user.SetEnrollmentCertificate(userJSON.EnrollmentCertificate)
	key, err := c.cryptoSuite.GetKey(userJSON.PrivateKeySKI)
	if err != nil {
		return nil, fmt.Errorf("cryptoSuite GetKey return error: %v", err)
	}
	user.SetPrivateKey(key)
	c.userContext = user
	return c.userContext, nil

}

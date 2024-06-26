/*
 * Copyright (c) 2024 OceanBase.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package secure

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/oceanbase/obshell/agent/constant"
	"github.com/oceanbase/obshell/agent/lib/crypto"
	"github.com/oceanbase/obshell/agent/meta"
)

type HttpHeader struct {
	Auth         string
	Ts           string
	Token        string
	Uri          string
	Keys         []byte
	IsForword    bool
	ForwardAgent meta.AgentInfo
}

func BuildHeader(agentInfo meta.AgentInfoInterface, uri string, isForword bool, keys ...[]byte) map[string]string {
	headers := make(map[string]string)
	pk := GetAgentPublicKey(agentInfo)
	if pk == "" {
		log.Warnf("no key for agent '%s:%d'", agentInfo.GetIp(), agentInfo.GetPort())
		return nil
	}

	var token string
	if isForword && !meta.OCS_AGENT.IsMasterAgent() {
		token, _ = getTokenByAgentInfo(meta.OCS_AGENT)
	} else {
		token, _ = getTokenByAgentInfo(agentInfo)
	}

	var aesKeys []byte
	if len(keys) != 2 {
		aesKeys = nil
	} else {
		aesKeys = append(keys[0], keys[1]...)
	}
	header := HttpHeader{
		Auth:  meta.OCEANBASE_PWD,
		Ts:    fmt.Sprintf("%d", time.Now().Add(getAuthExpiredDuration()).Unix()),
		Token: token,
		Uri:   uri,
		Keys:  aesKeys,
	}
	if isForword {
		header.IsForword = true
		header.ForwardAgent = meta.OCS_AGENT.GetAgentInfo()
	}

	mAuth, err := json.Marshal(header)
	if err != nil {
		log.WithError(err).Error("json marshal failed")
		return nil
	}
	auth, err := crypto.RSAEncrypt(mAuth, pk)
	if err != nil {
		log.WithError(err).Error("rsa encrypt failed")
		return nil
	}
	headers[constant.OCS_HEADER] = auth
	return headers
}

func DecryptHeader(ciphertext string) (HttpHeader, error) {
	decHeader, _ := Crypter.DecryptAndReturnBytes(ciphertext)
	var headers HttpHeader
	err := json.Unmarshal(decHeader, &headers)
	if err != nil {
		return headers, err
	}
	return headers, nil
}

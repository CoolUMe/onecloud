// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerBackendgroupResourceBase struct {
	// 负载均衡后端组ID
	BackendGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"backend_group_id"`
}

type SLoadbalancerBackendgroupResourceBaseManager struct {
	SLoadbalancerResourceBaseManager
}

func ValidateLoadbalancerBackendgroupResourceInput(userCred mcclient.TokenCredential, input api.LoadbalancerBackendGroupResourceInput) (*SLoadbalancerBackendGroup, api.LoadbalancerBackendGroupResourceInput, error) {
	lbbgObj, err := LoadbalancerBackendGroupManager.FetchByIdOrName(userCred, input.BackendGroup)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", LoadbalancerBackendGroupManager.Keyword(), input.BackendGroup)
		} else {
			return nil, input, errors.Wrap(err, "LoadbalancerBackendGroupManager.FetchByIdOrName")
		}
	}
	input.BackendGroup = lbbgObj.GetId()
	return lbbgObj.(*SLoadbalancerBackendGroup), input, nil
}

func (self *SLoadbalancerBackendgroupResourceBase) GetLoadbalancerBackendGroup() *SLoadbalancerBackendGroup {
	w, _ := LoadbalancerBackendGroupManager.FetchById(self.BackendGroupId)
	if w != nil {
		return w.(*SLoadbalancerBackendGroup)
	}
	return nil
}

func (self *SLoadbalancerBackendgroupResourceBase) GetLoadbalancer() *SLoadbalancer {
	lbbg := self.GetLoadbalancerBackendGroup()
	if lbbg != nil {
		return lbbg.GetLoadbalancer()
	}
	return nil
}

func (self *SLoadbalancerBackendgroupResourceBase) GetVpc() *SVpc {
	lb := self.GetLoadbalancer()
	if lb != nil {
		return lb.GetVpc()
	}
	return nil
}

func (self *SLoadbalancerBackendgroupResourceBase) GetCloudprovider() *SCloudprovider {
	lb := self.GetLoadbalancer()
	if lb != nil {
		return lb.GetCloudprovider()
	}
	return nil
}

func (self *SLoadbalancerBackendgroupResourceBase) GetProviderName() string {
	lb := self.GetLoadbalancer()
	if lb != nil {
		return lb.SManagedResourceBase.GetProviderName()
	}
	return ""
}

func (self *SLoadbalancerBackendgroupResourceBase) GetRegion() *SCloudregion {
	vpc := self.GetVpc()
	if vpc == nil {
		return nil
	}
	region, _ := vpc.GetRegion()
	return region
}

func (self *SLoadbalancerBackendgroupResourceBase) GetZone() *SZone {
	lb := self.GetLoadbalancer()
	if lb != nil {
		return lb.GetZone()
	}
	return nil
}

func (self *SLoadbalancerBackendgroupResourceBase) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) api.LoadbalancerBackendGroupResourceInfo {
	return api.LoadbalancerBackendGroupResourceInfo{}
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerBackendGroupResourceInfo {
	rows := make([]api.LoadbalancerBackendGroupResourceInfo, len(objs))

	lbbgIds := make([]string, len(objs))
	for i := range objs {
		var base *SLoadbalancerBackendgroupResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			lbbgIds[i] = base.BackendGroupId
		}
	}

	lbbgs := make(map[string]SLoadbalancerBackendGroup)
	err := db.FetchStandaloneObjectsByIds(LoadbalancerBackendGroupManager, lbbgIds, &lbbgs)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	lbList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.LoadbalancerBackendGroupResourceInfo{}
		if lbbg, ok := lbbgs[lbbgIds[i]]; ok {
			rows[i].BackendGroup = lbbg.Name
			rows[i].LoadbalancerId = lbbg.LoadbalancerId
		}
		lbList[i] = &SLoadbalancerResourceBase{rows[i].LoadbalancerId}
	}

	lbRows := manager.SLoadbalancerResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, lbList, fields, isList)

	for i := range rows {
		rows[i].LoadbalancerResourceInfo = lbRows[i]
	}
	return rows
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendGroupFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.BackendGroup) > 0 {
		lbbgObj, _, err := ValidateLoadbalancerBackendgroupResourceInput(userCred, query.LoadbalancerBackendGroupResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateLoadbalancerBackendgroupResourceInput")
		}
		q = q.Equals("backend_group_id", lbbgObj.GetId())
	}

	lbbgQ := LoadbalancerBackendGroupManager.Query("id").Snapshot()

	lbbgQ, err := manager.SLoadbalancerResourceBaseManager.ListItemFilter(ctx, lbbgQ, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemFilter")
	}

	if lbbgQ.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("backend_group_id"), lbbgQ.SubQuery()))
	}
	return q, nil
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "backend_group" {
		lbbgQuery := LoadbalancerBackendGroupManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(lbbgQuery.Field("name", field))
		q = q.Join(lbbgQuery, sqlchemy.Equals(q.Field("backend_group_id"), lbbgQuery.Field("id")))
		q.GroupBy(lbbgQuery.Field("name"))
		return q, nil
	} else {
		lbbgs := LoadbalancerBackendGroupManager.Query("id", "loadbalancer_id").SubQuery()
		q = q.LeftJoin(lbbgs, sqlchemy.Equals(q.Field("backend_id"), lbbgs.Field("id")))
		q, err := manager.SLoadbalancerResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendGroupFilterListInput,
) (*sqlchemy.SQuery, error) {
	q, orders, fields := manager.GetOrderBySubQuery(q, userCred, query)
	if len(orders) > 0 {
		q = db.OrderByFields(q, orders, fields)
	}
	return q, nil
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendGroupFilterListInput,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	lbbgQ := LoadbalancerBackendGroupManager.Query("id", "name")
	var orders []string
	var fields []sqlchemy.IQueryField

	if db.NeedOrderQuery(manager.SLoadbalancerResourceBaseManager.GetOrderByFields(query.LoadbalancerFilterListInput)) {
		var lbOrders []string
		var lbFields []sqlchemy.IQueryField
		lbbgQ, lbOrders, lbFields = manager.SLoadbalancerResourceBaseManager.GetOrderBySubQuery(lbbgQ, userCred, query.LoadbalancerFilterListInput)
		if len(lbOrders) > 0 {
			orders = append(orders, lbOrders...)
			fields = append(fields, lbFields...)
		}
	}

	if db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		subq := lbbgQ.SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("backend_group_id"), subq.Field("id")))
		if db.NeedOrderQuery([]string{query.OrderByBackendGroup}) {
			orders = append(orders, query.OrderByBackendGroup)
			fields = append(fields, subq.Field("name"))
		}
	}

	return q, orders, fields
}

func (manager *SLoadbalancerBackendgroupResourceBaseManager) GetOrderByFields(query api.LoadbalancerBackendGroupFilterListInput) []string {
	fields := make([]string, 0)
	lbFields := manager.SLoadbalancerResourceBaseManager.GetOrderByFields(query.LoadbalancerFilterListInput)
	fields = append(fields, lbFields...)
	fields = append(fields, query.OrderByBackendGroup)
	return fields
}
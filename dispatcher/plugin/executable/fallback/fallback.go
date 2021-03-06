//     Copyright (C) 2020-2021, IrineSistiana
//
//     This file is part of mosdns.
//
//     mosdns is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     mosdns is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package fallback

import (
	"context"
	"github.com/IrineSistiana/mosdns/dispatcher/handler"
	"github.com/IrineSistiana/mosdns/dispatcher/utils"
)

const PluginType = "fallback"

func init() {
	handler.RegInitFunc(PluginType, Init, func() interface{} { return new(Args) })
}

var _ handler.ExecutablePlugin = (*fallback)(nil)

type fallback struct {
	*handler.BP

	fallbackECS *utils.FallbackECS
}

type Args = utils.FallbackConfig

func Init(bp *handler.BP, args interface{}) (p handler.Plugin, err error) {
	return newFallback(bp, args.(*Args))
}

func newFallback(bp *handler.BP, args *Args) (*fallback, error) {
	fallbackECS, err := utils.ParseFallbackECS(args)
	if err != nil {
		return nil, err
	}
	return &fallback{
		BP:          bp,
		fallbackECS: fallbackECS,
	}, nil
}

func (f *fallback) Exec(ctx context.Context, qCtx *handler.Context) (err error) {
	return utils.WalkExecutableCmd(ctx, qCtx, f.L(), f.fallbackECS)
}

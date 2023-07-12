package graphql

import (
	"context"

	"github.com/Top-Weerapat-Mungmee/api-go-starter/models"
)

// Room returns RoomResolver implementation.
func (r *Resolver) Room() RoomResolver { return &roomResolver{r} }

type roomResolver struct{ *Resolver }

func (r *roomResolver) Devices(ctx context.Context, obj *models.Room) ([]*models.Device, error) {
	return r.DeviceRepo.GetDevicesByRoomID(obj.ID)
}

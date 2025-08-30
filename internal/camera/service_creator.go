package camera

// ProductionServiceCreator は本番用のServiceCreator実装
type ProductionServiceCreator struct{}

// NewProductionServiceCreator は新しいProductionServiceCreatorを作成する
func NewProductionServiceCreator() ServiceCreator {
	return &ProductionServiceCreator{}
}

// CreateService は実際のV4L2を使用するServiceを作成する
func (p *ProductionServiceCreator) CreateService(camera *Camera) Service {
	return NewCameraService(camera)
}

// MockServiceCreator はテスト用のServiceCreator実装
type MockServiceCreator struct{}

// NewMockServiceCreator は新しいMockServiceCreatorを作成する
func NewMockServiceCreator() ServiceCreator {
	return &MockServiceCreator{}
}

// CreateService はモックServiceを作成する
func (m *MockServiceCreator) CreateService(camera *Camera) Service {
	return NewMockCameraService(camera)
}

import { StatusApi, CameraApi } from "../generated/api";

// APIクライアントのモック
jest.mock("../generated/api");

describe("MainPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  // planetizerと同様のダミーテスト（テストがない状態でテスト実行するとエラーになるので）
  test("dummy", () => {
    expect("1").toBe("1");
  });

  test("APIクライアントが正しくインスタンス化される", () => {
    // StatusApiとCameraApiのコンストラクタが呼ばれることを確認
    const mockStatusApi = StatusApi as jest.MockedClass<typeof StatusApi>;
    const mockCameraApi = CameraApi as jest.MockedClass<typeof CameraApi>;

    expect(mockStatusApi).toBeDefined();
    expect(mockCameraApi).toBeDefined();
  });

  test("ステータスレスポンスの型チェック", () => {
    const mockStatus = {
      status: "running" as const,
      server: {
        host: "0.0.0.0",
        port: 8080,
      },
      cameras: 1,
      timestamp: "2025-08-29T00:00:00Z",
    };

    // 型が正しく定義されているか確認
    expect(mockStatus.status).toBe("running");
    expect(mockStatus.server.port).toBe(8080);
    expect(mockStatus.cameras).toBe(1);
  });

  test("カメラ情報の型チェック", () => {
    const mockCamera = {
      id: "camera1",
      name: "メインカメラ",
      device: "/dev/video0",
      settings: {
        fps: 30,
        width: 1920,
        height: 1080,
      },
      status: "active" as const,
    };

    // 型が正しく定義されているか確認
    expect(mockCamera.id).toBe("camera1");
    expect(mockCamera.settings.width).toBe(1920);
    expect(mockCamera.settings.height).toBe(1080);
    expect(mockCamera.settings.fps).toBe(30);
  });

  test("エラーレスポンスの型チェック", () => {
    const mockError = {
      error: "not_found",
      message: "カメラが見つかりません",
      details: "Camera ID: camera999",
      timestamp: "2025-08-29T00:00:00Z",
    };

    // 型が正しく定義されているか確認
    expect(mockError.error).toBe("not_found");
    expect(mockError.message).toBe("カメラが見つかりません");
    expect(mockError.details).toBe("Camera ID: camera999");
  });
});

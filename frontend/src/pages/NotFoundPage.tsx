export function NotFoundPage() {
  return (
    <div
      style={{
        padding: "20px",
        textAlign: "center",
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
      }}
    >
      <h1>404 - ページが見つかりません</h1>
      <p>お探しのページは存在しません。</p>
      <a
        href="/"
        style={{
          color: "#007bff",
          textDecoration: "none",
          padding: "10px 20px",
          border: "1px solid #007bff",
          borderRadius: "4px",
          marginTop: "20px",
        }}
      >
        ホームに戻る
      </a>
    </div>
  );
}

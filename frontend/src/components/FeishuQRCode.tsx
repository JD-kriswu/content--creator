interface FeishuQRCodeProps {
  qrUrl: string
  status: 'waiting' | 'success' | 'error'
  onRefresh?: () => void
}

export function FeishuQRCode({ qrUrl, status, onRefresh }: FeishuQRCodeProps) {
  return (
    <div className="flex flex-col items-center gap-4">
      <div className="w-64 h-64 border rounded-lg flex items-center justify-center bg-white">
        {status === 'waiting' && (
          <img src={qrUrl} alt="飞书扫码绑定" className="w-60 h-60" />
        )}
        {status === 'success' && (
          <div className="text-green-500 text-4xl">✅</div>
        )}
        {status === 'error' && (
          <div className="text-red-500 text-4xl">❌</div>
        )}
      </div>
      {status === 'waiting' && (
        <p className="text-gray-500 text-sm">请使用飞书 App 扫描二维码</p>
      )}
      {status === 'error' && onRefresh && (
        <button onClick={onRefresh} className="btn btn-secondary">
          刷新二维码
        </button>
      )}
    </div>
  )
}
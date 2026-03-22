import { useState } from "react";
import { useNavigate } from "react-router";
import { Card } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Separator } from "../components/ui/separator";
import { Phone, Lock, User, Feather } from "lucide-react";
import { toast } from "sonner";

export function Auth() {
  const [isLoading, setIsLoading] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [phoneNumber, setPhoneNumber] = useState("");
  const [verifyCode, setVerifyCode] = useState("");
  const navigate = useNavigate();

  const handlePhoneAuth = (e: React.FormEvent) => {
    e.preventDefault();
    
    // 验证手机号格式
    if (!/^1[3-9]\d{9}$/.test(phoneNumber)) {
      toast.error("请输入正确的手机号格式");
      return;
    }
    
    // 验证验证码
    if (verifyCode.length !== 6) {
      toast.error("请输入6位验证码");
      return;
    }
    
    // 测试账号验证
    if (phoneNumber === "13800138000" && verifyCode === "123456") {
      setIsLoading(true);
      setTimeout(() => {
        setIsLoading(false);
        toast.success("登录成功！欢迎回来");
        // 跳转到 dashboard
        navigate("/dashboard");
      }, 1500);
    } else {
      toast.error("手机号或验证码错误，请使用测试账号登录");
    }
  };

  const handleSendCode = () => {
    if (countdown > 0) return;
    
    setCountdown(60);
    toast.success("验证码已发送");
    
    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  const handleWechatAuth = () => {
    setIsLoading(true);
    setTimeout(() => {
      setIsLoading(false);
      toast.success("使用微信登录成功！");
    }, 1500);
  };

  return (
    <div className="max-w-md mx-auto">
      <div className="text-center mb-8">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl mb-4 shadow-xl shadow-blue-200">
          <Feather className="w-8 h-8 text-white" strokeWidth={2.5} />
        </div>
        <h1 className="text-3xl mb-2">欢迎来到轻写Claw</h1>
        <p className="text-gray-600">登录后可保存您的创作记录</p>
      </div>

      <Card className="p-6 sm:p-8 shadow-lg border-0 bg-white/80 backdrop-blur">
        {/* 测试账号提示 */}
        <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <div className="flex items-start gap-2">
            <div className="text-blue-600 mt-0.5">🔑</div>
            <div className="flex-1">
              <p className="text-sm font-medium text-blue-900 mb-1">测试账号</p>
              <p className="text-xs text-blue-700">
                手机号：<span className="font-mono font-semibold">13800138000</span>
              </p>
              <p className="text-xs text-blue-700">
                验证码：<span className="font-mono font-semibold">123456</span>
              </p>
            </div>
          </div>
        </div>

        <Tabs defaultValue="login" className="w-full">
          <TabsList className="grid w-full grid-cols-2 mb-6">
            <TabsTrigger value="login">登录</TabsTrigger>
            <TabsTrigger value="register">注册</TabsTrigger>
          </TabsList>

          <TabsContent value="login">
            <form onSubmit={handlePhoneAuth} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="login-phone">手机号</Label>
                <div className="relative">
                  <Phone className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input
                    id="login-phone"
                    type="tel"
                    placeholder="请输入手机号"
                    className="pl-10"
                    required
                    value={phoneNumber}
                    onChange={(e) => setPhoneNumber(e.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="login-code">验证码</Label>
                <div className="flex gap-2">
                  <div className="relative flex-1">
                    <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                    <Input
                      id="login-code"
                      type="text"
                      placeholder="请输入验证码"
                      className="pl-10"
                      required
                      value={verifyCode}
                      onChange={(e) => setVerifyCode(e.target.value)}
                    />
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleSendCode}
                    disabled={countdown > 0}
                    className="whitespace-nowrap"
                  >
                    {countdown > 0 ? `${countdown}秒` : "获取验证码"}
                  </Button>
                </div>
              </div>

              <Button
                type="submit"
                disabled={isLoading}
                className="w-full h-11 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
              >
                {isLoading ? "登录中..." : "登录"}
              </Button>
            </form>
          </TabsContent>

          <TabsContent value="register">
            <form onSubmit={handlePhoneAuth} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="register-name">用户名</Label>
                <div className="relative">
                  <User className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input
                    id="register-name"
                    type="text"
                    placeholder="请输入用户名"
                    className="pl-10"
                    required
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="register-phone">手机号</Label>
                <div className="relative">
                  <Phone className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input
                    id="register-phone"
                    type="tel"
                    placeholder="请输入手机号"
                    className="pl-10"
                    required
                    value={phoneNumber}
                    onChange={(e) => setPhoneNumber(e.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="register-code">验证码</Label>
                <div className="flex gap-2">
                  <div className="relative flex-1">
                    <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                    <Input
                      id="register-code"
                      type="text"
                      placeholder="请输入验证码"
                      className="pl-10"
                      required
                      value={verifyCode}
                      onChange={(e) => setVerifyCode(e.target.value)}
                    />
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleSendCode}
                    disabled={countdown > 0}
                    className="whitespace-nowrap"
                  >
                    {countdown > 0 ? `${countdown}秒` : "获取验证码"}
                  </Button>
                </div>
              </div>

              <Button
                type="submit"
                disabled={isLoading}
                className="w-full h-11 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
              >
                {isLoading ? "注册中..." : "注册"}
              </Button>
            </form>
          </TabsContent>
        </Tabs>

        <div className="mt-6">
          <div className="relative">
            <Separator />
            <span className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-white px-2 text-xs text-gray-500">
              或使用以下方式
            </span>
          </div>

          <div className="mt-6">
            <Button
              variant="outline"
              onClick={handleWechatAuth}
              disabled={isLoading}
              className="w-full h-11 border-gray-300 hover:border-green-600 hover:text-green-600"
            >
              <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="currentColor">
                <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 0 1 .213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 0 0 .167-.054l1.903-1.114a.864.864 0 0 1 .717-.098 10.16 10.16 0 0 0 2.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178A1.17 1.17 0 0 1 4.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178 1.17 1.17 0 0 1-1.162-1.178c0-.651.52-1.18 1.162-1.18zm5.34 2.867c-1.797-.052-3.746.512-5.28 1.786-1.72 1.428-2.687 3.72-1.78 6.22.942 2.453 3.666 4.229 6.884 4.229.826 0 1.622-.12 2.361-.336a.722.722 0 0 1 .598.082l1.584.926a.272.272 0 0 0 .14.047c.134 0 .24-.111.24-.247 0-.06-.023-.12-.038-.177l-.327-1.233a.582.582 0 0 1 .088-.46c1.532-1.163 2.517-2.908 2.517-4.848 0-3.27-2.72-5.908-6.24-5.908l-.747-.01zm-2.545 3.833c.535 0 .969.44.969.982a.976.976 0 0 1-.969.983.976.976 0 0 1-.969-.983c0-.542.434-.982.969-.982zm4.844 0c.535 0 .969.44.969.982a.976.976 0 0 1-.969.983.976.976 0 0 1-.969-.983c0-.542.434-.982.969-.982z"/>
              </svg>
              微信登录
            </Button>
          </div>
        </div>

        <p className="text-xs text-center text-gray-500 mt-6">
          登录即表示您同意我们的
          <a href="#" className="text-blue-600 hover:underline">服务条款</a>
          和
          <a href="#" className="text-blue-600 hover:underline">隐私政策</a>
        </p>
      </Card>

      <div className="mt-6 text-center text-sm text-gray-500">
        <p>💡 提示：登录后可同步检测记录到多设备</p>
      </div>
    </div>
  );
}
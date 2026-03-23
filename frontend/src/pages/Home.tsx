import { Link } from "react-router";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { 
  Feather, 
  Zap, 
  CheckCircle2,
  Users,
  Brain,
  MessageSquare,
  TrendingUp,
  Clock,
  Target
} from "lucide-react";

export function Home() {
  return (
    <div className="h-full overflow-y-auto">
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Hero Section - 第一屏 */}
      <div className="grid lg:grid-cols-2 gap-12 items-center mb-24">
        <div>
          <h1 className="text-4xl sm:text-5xl lg:text-6xl mb-6 leading-tight">
            <span className="text-gray-900 font-bold">一键生成</span>
            <span className="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent font-bold">爆款文案</span>
          </h1>
          
          <p className="text-xl text-gray-600 mb-12 leading-relaxed">
            你的AI文案助手
          </p>
          
          {/* 三大痛点解决 */}
          <div className="space-y-4 mb-8">
            <div className="flex items-start gap-3">
              <div className="w-6 h-6 bg-blue-100 rounded-full flex items-center justify-center flex-shrink-0 mt-1">
                <CheckCircle2 className="w-4 h-4 text-blue-600" />
              </div>
              <div>
                <span className="text-gray-700 font-medium">创作效率低？</span>
                <span className="text-gray-900"> → 5分钟帮你生成高质量文稿</span>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="w-6 h-6 bg-purple-100 rounded-full flex items-center justify-center flex-shrink-0 mt-1">
                <CheckCircle2 className="w-4 h-4 text-purple-600" />
              </div>
              <div>
                <span className="text-gray-700 font-medium">担心平台查重？</span>
                <span className="text-gray-900"> → 结合你的观点拓展，保证原创</span>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="w-6 h-6 bg-green-100 rounded-full flex items-center justify-center flex-shrink-0 mt-1">
                <CheckCircle2 className="w-4 h-4 text-green-600" />
              </div>
              <div>
                <span className="text-gray-700 font-medium">风格不统一？</span>
                <span className="text-gray-900"> → 越用越懂你的个性化风格</span>
              </div>
            </div>
          </div>
          
          <div className="flex flex-wrap gap-4 mb-8">
            <Link to="/auth">
              <Button 
                className="h-12 px-8 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white text-base"
              >
                <Feather className="w-5 h-5 mr-2" />
                免费试用，每日登录送200积分
              </Button>
            </Link>
          </div>
        </div>
        
        {/* 右侧数据卡片 */}
        <div className="hidden lg:block">
          <div className="relative">
            <div className="absolute inset-0 bg-gradient-to-r from-blue-400 to-purple-400 rounded-3xl blur-3xl opacity-20"></div>
            <Card className="relative p-8 shadow-2xl border-0 bg-white">
              <div className="grid grid-cols-2 gap-4">
                <div className="p-4 bg-gradient-to-br from-blue-50 to-blue-100 rounded-xl">
                  <Clock className="w-8 h-8 text-blue-600 mb-3" />
                  <div className="text-2xl font-bold text-gray-900">5-8分钟</div>
                  <div className="text-sm text-gray-600">平均创作时间</div>
                </div>
                <div className="p-4 bg-gradient-to-br from-purple-50 to-purple-100 rounded-xl">
                  <Target className="w-8 h-8 text-purple-600 mb-3" />
                  <div className="text-2xl font-bold text-gray-900">&lt;30%</div>
                  <div className="text-sm text-gray-600">相似度保障</div>
                </div>
                <div className="p-4 bg-gradient-to-br from-green-50 to-green-100 rounded-xl">
                  <TrendingUp className="w-8 h-8 text-green-600 mb-3" />
                  <div className="text-2xl font-bold text-gray-900">92%</div>
                  <div className="text-sm text-gray-600">用户满意度</div>
                </div>
                <div className="p-4 bg-gradient-to-br from-orange-50 to-orange-100 rounded-xl">
                  <Brain className="w-8 h-8 text-orange-600 mb-3" />
                  <div className="text-2xl font-bold text-gray-900">越用越懂你</div>
                  <div className="text-sm text-gray-600">智能进化</div>
                </div>
              </div>
            </Card>
          </div>
        </div>
      </div>

      {/* 第二屏：你能获得什么 */}
      <div className="mb-24">
        <div className="text-center mb-12">
          <h2 className="text-3xl font-bold text-gray-900 mb-4">🎯 你能获得什么</h2>
        </div>
        
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
          <Card className="p-6 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="text-center">
              <div className="text-3xl mb-3">✅</div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">日更无忧</h3>
              <p className="text-gray-600 text-sm">
                快速产出每日内容，告别创作瓶颈
              </p>
            </div>
          </Card>
          
          <Card className="p-6 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="text-center">
              <div className="text-3xl mb-3">✅</div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">质量保障</h3>
              <p className="text-gray-600 text-sm">
                专业级内容质量，爆款率提升3倍
              </p>
            </div>
          </Card>
          
          <Card className="p-6 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="text-center">
              <div className="text-3xl mb-3">✅</div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">风格沉淀</h3>
              <p className="text-gray-600 text-sm">
                建立个人内容品牌，形成独特风格
              </p>
            </div>
          </Card>
          
          <Card className="p-6 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="text-center">
              <div className="text-3xl mb-3">✅</div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">持续进化</h3>
              <p className="text-gray-600 text-sm">
                越用越聪明的AI助手，每次创作都更懂你
              </p>
            </div>
          </Card>
        </div>
      </div>

      {/* 第三屏：痛点解决方案 */}
      <div className="mb-24">
        <div className="text-center mb-12">
          <h2 className="text-3xl font-bold text-gray-900 mb-4">💡 为什么选择轻写Claw</h2>
        </div>
        
        <div className="grid md:grid-cols-3 gap-8">
          <Card className="p-8 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="mb-6">
              <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl flex items-center justify-center mb-4">
                <Zap className="w-6 h-6 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">创作效率低</h3>
            </div>
            
            <div className="space-y-4">
              <div className="flex items-start gap-2">
                <span className="text-lg">🤯</span>
                <div>
                  <div className="text-sm text-gray-500 mb-1">痛点</div>
                  <p className="text-gray-700">每天想选题、写稿子，耗时又耗力</p>
                </div>
              </div>
              
              <div className="flex items-start gap-2">
                <span className="text-lg">✅</span>
                <div>
                  <div className="text-sm text-gray-500 mb-1">解决方案</div>
                  <p className="text-gray-700">AI 5分钟生成高质量稿件</p>
                </div>
              </div>
              
              <div className="flex items-start gap-2">
                <span className="text-lg">🎯</span>
                <div>
                  <div className="text-sm text-gray-500 mb-1">结果</div>
                  <p className="text-gray-900 font-medium">日更无忧，创作效率提升10倍</p>
                </div>
              </div>
            </div>
          </Card>
          
          <Card className="p-8 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="mb-6">
              <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-purple-600 rounded-xl flex items-center justify-center mb-4">
                <Target className="w-6 h-6 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">担心平台查重</h3>
            </div>
            
            <div className="space-y-4">
              <div className="flex items-start gap-2">
                <span className="text-lg">😰</span>
                <div>
                  <div className="text-sm text-gray-500 mb-1">痛点</div>
                  <p className="text-gray-700">内容雷同被限流，原创难保证</p>
                </div>
              </div>
              
              <div className="flex items-start gap-2">
                <span className="text-lg">✅</span>
                <div>
                  <div className="text-sm text-gray-500 mb-1">解决方案</div>
                  <p className="text-gray-700">严格相似度检测&lt;30%</p>
                </div>
              </div>
              
              <div className="flex items-start gap-2">
                <span className="text-lg">🎯</span>
                <div>
                  <div className="text-sm text-gray-500 mb-1">结果</div>
                  <p className="text-gray-900 font-medium">平台查重无忧，流量稳定增长</p>
                </div>
              </div>
            </div>
          </Card>
          
          <Card className="p-8 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="mb-6">
              <div className="w-12 h-12 bg-gradient-to-br from-green-500 to-green-600 rounded-xl flex items-center justify-center mb-4">
                <Brain className="w-6 h-6 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">风格不统一</h3>
            </div>
            
            <div className="space-y-4">
              <div className="flex items-start gap-2">
                <span className="text-lg">😕</span>
                <div>
                  <div className="text-sm text-gray-500 mb-1">痛点</div>
                  <p className="text-gray-700">多人创作或状态波动，风格杂乱</p>
                </div>
              </div>
              
              <div className="flex items-start gap-2">
                <span className="text-lg">✅</span>
                <div>
                  <div className="text-sm text-gray-500 mb-1">解决方案</div>
                  <p className="text-gray-700">越用越懂你的个性化风格</p>
                </div>
              </div>
              
              <div className="flex items-start gap-2">
                <span className="text-lg">🎯</span>
                <div>
                  <div className="text-sm text-gray-500 mb-1">结果</div>
                  <p className="text-gray-900 font-medium">建立个人品牌，粉丝识别度高</p>
                </div>
              </div>
            </div>
          </Card>
        </div>
      </div>

      {/* 第四屏：特色功能亮点 */}
      <div className="mb-24">
        <div className="text-center mb-12">
          <h2 className="text-3xl font-bold text-gray-900 mb-4">🎪 独特的AI创作能力</h2>
        </div>
        
        <div className="grid md:grid-cols-3 gap-6">
          <Card className="p-6 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl flex items-center justify-center mb-4">
              <Users className="w-6 h-6 text-white" />
            </div>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">6角色智能分析</h3>
            <p className="text-gray-600">
              6大AI专家协同工作，从爆款解构到风格匹配，全方位保障质量
            </p>
          </Card>
          
          <Card className="p-6 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-purple-600 rounded-xl flex items-center justify-center mb-4">
              <MessageSquare className="w-6 h-6 text-white" />
            </div>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">对话式智能引导</h3>
            <p className="text-gray-600">
              像聊天一样简单，实时展示思考过程，建立信任
            </p>
          </Card>
          
          <Card className="p-6 border-0 shadow-lg hover:shadow-xl transition-shadow bg-white">
            <div className="w-12 h-12 bg-gradient-to-br from-green-500 to-green-600 rounded-xl flex items-center justify-center mb-4">
              <Brain className="w-6 h-6 text-white" />
            </div>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">越用越聪明</h3>
            <p className="text-gray-600">
              学习你的风格偏好，建立专属知识库，每次创作都更懂你
            </p>
          </Card>
        </div>
      </div>

      {/* 第五屏：数据证明 */}
      <div className="mb-24">
        <div className="text-center mb-12">
          <h2 className="text-3xl font-bold text-gray-900 mb-4">📊 数据说话</h2>
        </div>
        
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
          <Card className="p-6 border-0 shadow-lg bg-gradient-to-br from-blue-50 to-blue-100">
            <div className="flex items-center gap-3 mb-2">
              <Clock className="w-6 h-6 text-blue-600" />
              <span className="text-sm text-gray-600">平均创作时间</span>
            </div>
            <div className="text-3xl font-bold text-gray-900 mb-1">5-8分钟/篇</div>
            <p className="text-sm text-gray-600">比手动快10倍</p>
          </Card>
          
          <Card className="p-6 border-0 shadow-lg bg-gradient-to-br from-purple-50 to-purple-100">
            <div className="flex items-center gap-3 mb-2">
              <Target className="w-6 h-6 text-purple-600" />
              <span className="text-sm text-gray-600">原创度保障</span>
            </div>
            <div className="text-3xl font-bold text-gray-900 mb-1">相似度&lt;30%</div>
            <p className="text-sm text-gray-600">平台查重无忧</p>
          </Card>
          
          <Card className="p-6 border-0 shadow-lg bg-gradient-to-br from-green-50 to-green-100">
            <div className="flex items-center gap-3 mb-2">
              <TrendingUp className="w-6 h-6 text-green-600" />
              <span className="text-sm text-gray-600">用户满意度</span>
            </div>
            <div className="text-3xl font-bold text-gray-900 mb-1">92%</div>
            <p className="text-sm text-gray-600">用户愿意持续使用</p>
          </Card>
          
          <Card className="p-6 border-0 shadow-lg bg-gradient-to-br from-orange-50 to-orange-100">
            <div className="flex items-center gap-3 mb-2">
              <Brain className="w-6 h-6 text-orange-600" />
              <span className="text-sm text-gray-600">智能进化</span>
            </div>
            <div className="text-3xl font-bold text-gray-900 mb-1">越用越懂你</div>
            <p className="text-sm text-gray-600">持续学习优化</p>
          </Card>
        </div>
      </div>

      {/* 第六屏：CTA 行动号召 */}
      <Card className="p-8 sm:p-12 text-center bg-gradient-to-r from-blue-600 to-purple-600 border-0 shadow-2xl">
        <div className="max-w-2xl mx-auto">
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            开始创作您的爆款口播稿
          </h2>
          <p className="text-blue-100 text-lg mb-8">
            加入10万+创作者，体验AI驱动的内容创作新方式
          </p>
          <Link to="/auth">
            <Button 
              className="h-14 px-10 bg-white text-blue-600 hover:bg-gray-50 text-lg font-medium"
            >
              <Feather className="w-5 h-5 mr-2" />
              立即开始试用
            </Button>
          </Link>
        </div>
      </Card>

      <div className="mt-12 text-center text-sm text-gray-500">
        <p>专为内容创作者打造 • 支持中英文 • 智能学习你的风格</p>
      </div>
    </div>
    </div>
  );
}
import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router";
import { ArrowUp, Sparkles } from "lucide-react";
import { MessageList, Message } from "../components/create/MessageList";
import { ChatInput } from "../components/create/ChatInput";
import { Sidebar } from "../components/Sidebar";
import { LoadingState } from "../components/create/LoadingState";
import { OutlineEditor } from "../components/create/OutlineEditor";
import { ScriptEditor } from "../components/create/ScriptEditor";

type Stage = 'initial' | 'generating-outline' | 'outline-confirm' | 'generating-script' | 'script-done';

interface ChatHistory {
  id: string;
  title: string;
  timestamp: Date;
  reference: string;
  messages: Message[];
  outlineText: string;
  script: string;
  stage: Stage;
}

export function Dashboard() {
  const [stage, setStage] = useState<Stage>('initial');
  const [reference, setReference] = useState("");
  const [messages, setMessages] = useState<Message[]>([]);
  const [outlineText, setOutlineText] = useState("");
  const [script, setScript] = useState("");
  const [currentChatId, setCurrentChatId] = useState<string>('');
  const [historyList, setHistoryList] = useState<ChatHistory[]>([
    {
      id: 'chat-1',
      title: '如何在30天内提升抖音粉丝10万',
      timestamp: new Date(2024, 2, 20, 14, 30),
      reference: '如何在30天内提升抖音粉丝10万...',
      messages: [],
      outlineText: '',
      script: '',
      stage: 'initial'
    },
    {
      id: 'chat-2',
      title: '小红书爆款文案创作技巧分享',
      timestamp: new Date(2024, 2, 19, 10, 15),
      reference: '小红书爆款文案创作技巧分享...',
      messages: [],
      outlineText: '',
      script: '',
      stage: 'initial'
    },
    {
      id: 'chat-3',
      title: '5分钟学会短视频文案写作',
      timestamp: new Date(2024, 2, 18, 16, 45),
      reference: '5分钟学会短视频文案写作...',
      messages: [],
      outlineText: '',
      script: '',
      stage: 'initial'
    },
    {
      id: 'chat-4',
      title: '电商直播话术模板大全',
      timestamp: new Date(2024, 2, 17, 9, 20),
      reference: '电商直播话术模板大全...',
      messages: [],
      outlineText: '',
      script: '',
      stage: 'initial'
    },
    {
      id: 'chat-5',
      title: '品牌营销文案如何打动用户',
      timestamp: new Date(2024, 2, 16, 13, 50),
      reference: '品牌营销文案如何打动用户...',
      messages: [],
      outlineText: '',
      script: '',
      stage: 'initial'
    }
  ]);
  const navigate = useNavigate();
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // 自动滚动到消息底部
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const addMessage = (type: Message['type'], content: string, action?: Message['action']) => {
    const newMessage: Message = {
      id: `msg-${Date.now()}-${Math.random()}`,
      type,
      content,
      timestamp: new Date(),
      action
    };
    setMessages(prev => [...prev, newMessage]);
  };

  const handleCreate = () => {
    if (!reference.trim()) return;
    
    // 切换到创作模式
    setStage('generating-outline');
    
    // 添加用户消息
    addMessage('user', reference);
    
    // AI 欢迎消息
    setTimeout(() => {
      addMessage('ai', '好的！我已经收到你的参考文案。让我先为你生成一个大纲...');
    }, 500);
    
    // 模拟生成大纲
    setTimeout(() => {
      const mockOutline = `开场引入：用一个震撼的问题或场景吸引注意力
痛点描述：深挖用户的真实困扰和需求
解决方案：提出核心价值和独特卖点
案例证明：用具体数据或故事增强说服力
行动召唤：引导用户立即采取行动`;
      
      setOutlineText(mockOutline);
      setStage('outline-confirm');
      addMessage('system', '✅ 大纲已生成！请在右侧查看和编辑', {
        label: '确认大纲，生成完整稿',
        onClick: handleConfirmOutline
      });
    }, 2500);
  };

  const handleConfirmOutline = () => {
    setStage('generating-script');
    addMessage('ai', '大纲确认完成！现在开始为你创作完整的爆款口播稿...');
    
    // 模拟生成完整稿
    setTimeout(() => {
      const mockScript = `你是不是也经历过这样的困境？每天花费大量时间写文案，却总是得不到理想的效果？

传统的文案创作不仅耗时耗力，而且很难保证原创度。你可能花了几个小时精心打磨一篇文案，结果发现转化率惨不忍睹。更糟糕的是，灵感枯竭的时候，看着空白的屏幕发呆，却怎么也写不出一个字。

现在，轻写Claw 来帮你解决这个问题！我们的 AI 文案助手不仅能快速生成高质量文案，还能学习你的风格，越用越懂你。只需要粘贴一段参考文案，AI 就能为你创作出全新的爆款内容。

已经有超过 10,000 名创作者在使用轻写Claw。他们平均每天节省 2 小时的文案创作时间，文案转化率提升 35%。一位用户说："以前我一天最多写 3 条文案，现在一天能产出 20 条，而且质量更高了！"

还在等什么？立即开始你的第一次创作，让 AI 成为你的文案搭档！点击下方按钮，免费体验轻写Claw 的强大功能！`;
      
      setScript(mockScript);
      setStage('script-done');
      addMessage('system', '🎉 完整口播稿已生成！你可以在右侧直接编辑和导出');
    }, 3500);
  };

  const handleSendMessage = (content: string) => {
    addMessage('user', content);
    
    // 模拟 AI 回复
    setTimeout(() => {
      addMessage('ai', '好的，我理解你的需求。让我根据你的反馈进行调整...');
    }, 1000);
  };

  const handleRegenerateScript = () => {
    setStage('generating-script');
    addMessage('ai', '好的，我将重新生成口播稿...');
    
    setTimeout(() => {
      setStage('script-done');
      addMessage('system', '✅ 口播稿已重新生成');
    }, 3000);
  };

  // 返回首页
  const handleBack = () => {
    setStage('initial');
    setReference('');
    setMessages([]);
    setOutlineText('');
    setScript('');
  };

  // 开始新对话（自动保存当前内容）
  const handleNewChat = () => {
    // 保存当前对话到历史记录
    if (messages.length > 0 || outlineText || script) {
      const chatId = currentChatId || `chat-${Date.now()}`;
      const title = reference.slice(0, 30) + (reference.length > 30 ? '...' : '');
      
      const newHistory: ChatHistory = {
        id: chatId,
        title,
        timestamp: new Date(),
        reference,
        messages,
        outlineText,
        script,
        stage
      };
      
      setHistoryList(prev => [newHistory, ...prev]);
    }
    
    // 清空状态，开始新对话
    setCurrentChatId(`chat-${Date.now()}`);
    setMessages([]);
    setOutlineText('');
    setScript('');
    setStage('initial');
    setReference('');
  };

  // 加载历史对话
  const handleSelectHistory = (id: string) => {
    const history = historyList.find(h => h.id === id);
    if (!history) return;
    
    // 恢复对话状态
    setCurrentChatId(history.id);
    setReference(history.reference);
    setMessages(history.messages);
    setOutlineText(history.outlineText);
    setScript(history.script);
    setStage(history.stage);
  };

  // 初始状态：中间大输入框
  if (stage === 'initial') {
    return (
      <div className="h-screen flex overflow-hidden">
        {/* 侧边栏 */}
        <Sidebar 
          onNewChat={handleNewChat}
          historyItems={historyList.map(h => ({
            id: h.id,
            title: h.title,
            timestamp: h.timestamp
          }))}
          currentChatId={currentChatId}
          onSelectHistory={handleSelectHistory}
        />
        
        {/* 主内容区 */}
        <div className="flex-1 overflow-y-auto">
          <div className="max-w-4xl mx-auto px-4">
            {/* Hero 输入区 */}
            <div className="text-center mb-8 pt-12">
              <h1 className="text-4xl sm:text-5xl mb-3">Hi，今天想创作什么爆款文案？</h1>
              <p className="text-lg text-gray-500 mb-4">粘贴你的参考口播稿，AI 会学习风格并为你创作</p>
              <div className="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-blue-50 to-purple-50 rounded-full border border-blue-200">
                <Sparkles className="w-4 h-4 text-blue-600" />
                <span className="text-sm font-medium text-blue-900">越用越懂你</span>
              </div>
            </div>

            {/* 主输入框 */}
            <div className="mb-8">
              <div className="relative">
                <textarea
                  value={reference}
                  onChange={(e) => setReference(e.target.value)}
                  placeholder="粘贴你喜欢的爆款口播稿..."
                  className="w-full h-[200px] p-6 pr-20 pb-20 text-base border border-gray-200 rounded-2xl resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                />
                
                {/* 右下角圆角矩形提交按钮 */}
                <button
                  onClick={handleCreate}
                  disabled={!reference.trim() || reference.length < 10}
                  className="absolute bottom-4 right-4 w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center shadow-lg hover:scale-105 hover:shadow-xl transition-all disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:scale-100"
                >
                  <ArrowUp className="w-5 h-5 text-white" strokeWidth={2.5} />
                </button>
              </div>
              
              {/* 字数统计 */}
              <div className="mt-3 px-2">
                <p className="text-sm text-gray-400">
                  {reference.length} 字
                </p>
              </div>
            </div>

            {/* 底部提示 */}
            <div className="text-center pb-16 text-sm text-gray-400">
              <p>💡 提示：提供参考文案可以帮助 AI 更好地理解你想要的风格</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // 创作模式：左右分栏
  return (
    <div className="h-screen flex overflow-hidden">
      {/* 中间：对话交互区 */}
      <div className="w-full md:w-2/5 border-r border-gray-200 flex flex-col bg-white">
        {/* 消息列表 */}
        <MessageList messages={messages} />
        <div ref={messagesEndRef} />
        
        {/* 底部输入框 */}
        <ChatInput 
          onSend={handleSendMessage}
          placeholder="随时告诉我你的想法..."
          disabled={stage === 'generating-outline' || stage === 'generating-script'}
        />
      </div>

      {/* 右侧：内容生成区 */}
      <div className="hidden md:block md:w-3/5 h-full">
        {stage === 'generating-outline' && (
          <LoadingState message="正在分析并生成大纲..." />
        )}
        
        {stage === 'outline-confirm' && (
          <OutlineEditor content={outlineText} onChange={setOutlineText} />
        )}
        
        {stage === 'generating-script' && (
          <LoadingState message="正在创作爆款口播稿..." />
        )}
        
        {stage === 'script-done' && (
          <ScriptEditor 
            content={script}
            onChange={setScript}
            onRegenerate={handleRegenerateScript}
          />
        )}
      </div>
    </div>
  );
}
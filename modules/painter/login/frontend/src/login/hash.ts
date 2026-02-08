import encHex from 'crypto-js/enc-hex';
import MD5 from 'crypto-js/md5';
import { useEffect, useMemo, useState } from 'react';

/**
 * 使用方式
 * const { fingerprint, loading } = useDeviceFingerprint();
 */


/**
 * 获取音频指纹（基于 Web Audio API 的渲染差异）
 * @returns 音频指纹字符串（兜底返回 'no-audio'）
 */
const getAudioFingerprint = async (): Promise<string> => {
    try {
        // 兼容不同浏览器的 AudioContext 前缀
        const AudioContext = window.AudioContext || (window as any).webkitAudioContext;
        if (!AudioContext) return 'no-audio';

        // 创建音频上下文（部分浏览器需用户交互触发，这里做基础兼容）
        const audioCtx = new AudioContext();
        const oscillator = audioCtx.createOscillator();
        const analyser = audioCtx.createAnalyser();

        // 配置音频参数（固定参数放大渲染差异）
        oscillator.type = 'sine';
        oscillator.frequency.setValueAtTime(440, audioCtx.currentTime); // A4 音高
        analyser.fftSize = 2048;

        // 连接节点
        oscillator.connect(analyser);
        analyser.connect(audioCtx.destination);

        // 启动并快速停止振荡器（无需播放声音，仅需渲染数据）
        oscillator.start();
        oscillator.stop(audioCtx.currentTime + 0.001);

        // 获取频域数据（核心差异点）
        const bufferLength = analyser.frequencyBinCount;
        const dataArray = new Uint8Array(bufferLength);
        analyser.getByteFrequencyData(dataArray);

        // 关闭音频上下文，释放资源
        await audioCtx.close();

        // 转为字符串用于哈希
        return Array.from(dataArray).join(',');
    } catch (e) {
        // 无音频设备/权限/浏览器不支持时兜底
        return 'no-audio';
    }
};

/**
 * 获取 Canvas 指纹（复用之前的逻辑）
 */
const getCanvasFingerprint = (): string => {
    const canvas = document.createElement('canvas');
    canvas.width = 200;
    canvas.height = 100;
    const ctx = canvas.getContext('2d');
    if (!ctx) return 'no-canvas';

    // 绘制固定内容（触发渲染差异）
    ctx.fillStyle = '#f60';
    ctx.fillRect(10, 10, 100, 50);
    ctx.fillStyle = '#069';
    ctx.font = '18px Arial';
    ctx.fillText('Canvas Fingerprint', 20, 70);
    ctx.beginPath();
    ctx.arc(150, 50, 20, 0, Math.PI * 2, true);
    ctx.stroke();
    // 随机噪点放大差异
    for (let i = 0; i < 10; i++) {
        ctx.fillStyle = `rgba(${Math.random() * 255}, ${Math.random() * 255}, ${Math.random() * 255}, 0.5)`;
        ctx.fillRect(Math.random() * 200, Math.random() * 100, 1, 1);
    }

    // 提取像素数据
    const pixelData = ctx.getImageData(0, 0, canvas.width, canvas.height).data;
    return Array.from(pixelData).join(',');
};

/**
 * 多维度设备指纹 Hook
 * @returns { fingerprint: string, loading: boolean } 最终指纹 + 加载状态（音频指纹异步）
 */
export function useDeviceFingerprint() {
    const [fingerprint, setFingerprint] = useState<string>('');
    const [loading, setLoading] = useState<boolean>(true);

    // 固定维度信息（无需重复计算）
    const baseInfo = useMemo(() => {
        return {
            // 1. User-Agent 及浏览器基础信息
            userAgent: navigator.userAgent,
            platform: navigator.platform, // 操作系统平台（Win32/MacIntel等）
            language: navigator.language, // 浏览器语言（zh-CN/en-US等）
            timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone, // 时区（Asia/Shanghai等）

            // 2. 屏幕信息（含像素比，区分高清屏）
            screenWidth: screen.width,
            screenHeight: screen.height,
            devicePixelRatio: window.devicePixelRatio, // 像素比（1/2/3）
            colorDepth: screen.colorDepth, // 颜色深度（通常24/32）
            screenOrientation: screen.orientation?.type || 'unknown', // 屏幕方向

            // 3. 功能支持（辅助维度）
            maxTouchPoints: navigator.maxTouchPoints, // 触摸点数量（区分触屏/非触屏）
            cookieEnabled: navigator.cookieEnabled, // 是否启用Cookie
            webdriver: (navigator as any).webdriver || false, // 检测自动化工具（可选）
        };
    }, []);

    // 异步生成最终指纹
    useEffect(() => {
        const generateFingerprint = async () => {
            try {
                // 并行获取 Canvas + 音频指纹
                const [canvasFP, audioFP] = await Promise.all([
                    getCanvasFingerprint(),
                    getAudioFingerprint(),
                ]);

                // 整合所有维度为原始字符串（核心：所有差异维度拼接）
                const rawFingerprint = JSON.stringify({
                    ...baseInfo,
                    canvasFP,
                    audioFP,
                });

                // MD5 哈希生成最终指纹（32位固定长度）
                const finalFP = MD5(rawFingerprint).toString(encHex);
                setFingerprint(finalFP);
            } catch (e) {
                console.error('指纹生成失败:', e);
                setFingerprint('unknown');
            } finally {
                setLoading(false);
            }
        };

        generateFingerprint();
    }, [baseInfo]);

    return { fingerprint, loading };
}
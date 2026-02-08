import { api_login, api_vcode } from '@/apis/backend';
import { Button, Form, Input, Space, message } from 'antd'; // 新增：导入message做错误提示
import React, { useEffect, useState } from 'react';
import { useDeviceFingerprint } from './hash';
import styles from './login.module.css';

interface LoginFormValues {
    username: string;
    captcha: string;
}

const LoginForm: React.FC = () => {
    const [form] = Form.useForm();
    const [countdown, setCountdown] = useState<number>(0);
    const [uploading, setUploading] = useState<boolean>(false)
    const { fingerprint, loading: fingerprintLoading } = useDeviceFingerprint(); // 重命名：避免和其他loading混淆

    // 验证码倒计时逻辑
    useEffect(() => {
        let timer: NodeJS.Timeout | null = null;
        if (countdown > 0) {
            timer = setInterval(() => {
                setCountdown(prev => prev - 1);
            }, 1000);
        }
        return () => {
            if (timer) clearInterval(timer);
        };
    }, [countdown]);

    const handleGetCaptcha = async () => {
        try {
            // 优化：用表单内置校验，比直接getFieldValue更严谨，会触发原有必填规则
            const { username } = await form.validateFields(['username']);
            // 新增：指纹未加载完成时，不发起请求
            if (fingerprintLoading || !fingerprint) {
                message.warning('设备信息加载中，请稍候');
                return;
            }
            setCountdown(60);
            // 发起验证码请求
            await api_vcode({
                username: username,
                challengeResponse: "",
                device_id: fingerprint,
                device_info: navigator.userAgent
            });
            message.success('验证码发送成功，请查收');
        } catch (error) {
            // 新增：请求失败/校验失败时，清除倒计时，避免按钮一直禁用
            setCountdown(0);
            console.error('获取验证码失败：', error);
            message.error('获取验证码失败，请重试');
        }
    };

    const handleSubmit = async (_: LoginFormValues) => {
        try {
            setUploading(true);
            const { username, captcha } = form.getFieldsValue(); // 优化：一次性获取表单值，更简洁
            // 新增：指纹校验
            if (!fingerprint) {
                message.warning('设备信息加载失败，请刷新页面');
                setUploading(false);
                return;
            }
            const ret = await api_login({
                username: username,
                challengeResponse: captcha,
                device_id: fingerprint,
                device_token: navigator.userAgent,
            });
            console.log("登录结果：", ret);
            message.success('登录成功');
            // 登录成功后的逻辑，比如跳转到首页
            gotoSource()
        } catch (error) {
            console.error('登录失败：', error);
            message.error('登录失败，用户名或验证码错误');
        } finally {
            // 优化：用finally，无论成功失败都关闭loading
            setUploading(false);
        }
    };

    return (
        <div className={styles.maskWrapper}>
            <div className={styles.loginCard} >
                <Form
                    form={form}
                    layout="vertical"
                    onFinish={handleSubmit}
                    autoComplete="off"
                >
                    <Form.Item
                        name="username"
                        label="用户名"
                        rules={[{ required: true, message: '请输入用户名' }]}
                    >
                        <Input
                            placeholder="请输入用户名"
                            size="large"
                            autoFocus
                            className={styles.inputItem}
                        />
                    </Form.Item>

                    <Form.Item
                        name="captcha"
                        label="验证码"
                        rules={[{ required: true, message: '请输入验证码' }]}
                    >
                        <Space.Compact className={styles.captchaWrapper}>
                            <Input
                                placeholder="请输入6位验证码"
                                size="large"
                                // 核心修改：删掉disabled={countdown > 0}，解除输入框禁用
                                className={styles.inputItem}
                                maxLength={6} // 新增：限制6位输入，符合验证码规则
                            />
                            <Button
                                type="primary"
                                size="large"
                                onClick={handleGetCaptcha}
                                // 优化：指纹加载中也禁用按钮，避免无device_id请求
                                disabled={countdown > 0 || fingerprintLoading}
                            >
                                {countdown > 0 ? `${countdown}s后重新获取` : '获取验证码'}
                            </Button>
                        </Space.Compact>
                    </Form.Item>

                    <Form.Item>
                        <Button
                            type="primary"
                            htmlType="submit"
                            size="large"
                            className={styles.submitButton}
                            loading={fingerprintLoading || uploading} // 对应重命名后的loading
                        >
                            登录
                        </Button>
                    </Form.Item>
                </Form>
            </div>
        </div>
    );
};

function getQueryParam(key: string): string | null {
    // 解析当前URL的query参数（兼容hash模式和普通模式）
    const searchParams = new URLSearchParams(window.location.search);
    return searchParams.get(key);
}

function gotoSource() {
    const sourceUrl = getQueryParam('source');
    // 2. 定义默认跳转地址
    const defaultUrl = '/ui/';
    // 3. 最终跳转地址
    let redirectUrl = defaultUrl;

    // 4. 处理source参数：有值且合法则使用，否则用默认值
    if (sourceUrl) {
        // 解码URL（因为source的值是编码后的URL，需还原）
        const decodedSourceUrl = decodeURIComponent(sourceUrl);
        // 校验解码后的URL合法性
        if (isLegalUrl(decodedSourceUrl)) {
            redirectUrl = decodedSourceUrl;
        } else {
            console.warn('source参数值非法，使用默认跳转地址', decodedSourceUrl);
        }
    }

    // 5. 执行跳转（使用replace避免回退到当前页面）
    window.location.replace(redirectUrl);
}

function isLegalUrl(url: string): boolean {
    if (!url) return false;
    // 支持相对路径（如/ui/）和绝对路径（如https://xxx）
    try {
        // 相对路径会基于当前域名解析，绝对路径直接验证
        new URL(url, window.location.origin);
        return true;
    } catch (e) {
        return false;
    }
}

export default LoginForm;
import { message } from 'antd';
import axios, { AxiosError, AxiosResponse } from "axios";

const AXIOS = axios.create({
    baseURL: "/auth",
    timeout: 5000,
});

AXIOS.interceptors.response.use(
    // 成功响应：直接透传响应结果（业务接口自行处理2xx/3xx状态码）
    (response: AxiosResponse) => {
        return response;
    },
    // 失败响应：统一处理所有错误
    (error: AxiosError) => {
        const err = error;

        // 情况1：无response → 网络错误（断网、域名解析失败、服务器无法连接）
        if (!err.response) {
            handleNetworkErr(err);
        }
        // 情况2：有response → HTTP错误（4xx/5xx）
        else {
            const statusCode = err.response.status;
            // 子情况2.1：401/403/407 → 重定向到登录页
            if ([401, 403, 407].includes(statusCode)) {
                message.error("登录状态失效，请重新登录");
                console.log("权限错误详情：", err);
                // 重定向到登录页面（替换为你的实际登录地址）
                window.location.href = "https://www.baidu.com";
            }
            // 子情况2.2：其他4xx/5xx错误 → 提示+打印日志
            else {
                message.error(`请求失败：${statusCode} ${err.response.statusText}`);
                console.log("请求错误详情：", {
                    status: statusCode,
                    statusText: err.response.statusText,
                    errorMsg: err.message,
                    data: err.response.data
                });
            }
        }

        // 关键：处理完错误后，重新抛出错误（让业务接口的catch能捕获）
        return Promise.reject(error);
    }
);

function handleNetworkErr(err: AxiosError) {
    // 1. 判断：断网
    if (!navigator.onLine) {
        message.error("网络错误：设备已断网");
        console.log("断网错误详情：", err);
    }
    // 2. 判断：域名解析失败
    else if (err.code === "ENOTFOUND" || err.message.includes("DNS") || err.message.includes("无法解析主机") || err.message.includes("getaddrinfo ENOTFOUND")) {
        message.error("网络错误：域名解析失败");
        console.log("域名解析失败错误详情：", err);
    }
    // 3. 判断：服务器无法连接（排除上述两种后，剩余无response错误均为此类）
    else {
        // 细分超时和连接被拒绝
        let errorTip = "网络错误：服务器无法连接";
        if (err.code === "ECONNABORTED" || err.message.includes("Timeout")) {
            errorTip = "网络错误：请求超时";
        } else if (err.code === "ECONNREFUSED" || err.message.includes("Connection refused")) {
            errorTip = "网络错误：服务器连接被拒绝";
        }
        message.error(errorTip);
        console.log("服务器无法连接错误详情：", err);
    }
}

const RequestMap = {
    "获取用户信息": "/user",
    "登录": "/login",
    "获取验证码": "/v-code"
}

export interface loginRequest {
    username: string
    challengeResponse: string
    device_id: string
    device_token: string
}

export interface User {
    id: number
    ding_id: string
    updated_at: number
    role: string
    name: string
    email: string
    avatar_smail: string
    avatar_medium: string
    avatar_big: string
    favorite: {
        main_color: string[]
    }
}

//api_login 登录
export async function api_login(input: loginRequest): Promise<User | null> {
    const url = RequestMap["登录"]
    try {//2xx，3xx相关分支
        const ret = await AXIOS.post(url, input)
        switch (ret.status) {
            case 200: return ret.data as User
            default: return null //返回空数组表示需要使用缓存
        }
    } catch (error) {
        // 😄拦截器已处理错误提示和日志，这里只需返回默认值
        return null
    }
}

//api_getUser 已经登录情况下获取用户信息
export async function api_getUser(): Promise<User | null> {
    const url = RequestMap["获取用户信息"]
    try {//2xx，3xx相关分支
        const ret = await AXIOS.get(url)
        switch (ret.status) {
            case 200: return ret.data as User
            default: return null //返回空数组表示需要使用缓存
        }
    } catch (error) {
        // 😄拦截器已处理错误提示和日志，这里只需返回默认值
        return null
    }
}

interface vcodeRequest {
    username: string,
    challengeResponse: string,
    device_id: string,
    device_info: string
}

//api_vcode 获取验证码
export async function api_vcode(input: vcodeRequest): Promise<User | null> {
    const url = RequestMap["获取验证码"]
    try {//2xx，3xx相关分支
        const ret = await AXIOS.post(url, input)
        switch (ret.status) {
            case 200: return ret.data as User
            default: return null //返回空数组表示需要使用缓存
        }
    } catch (error) {
        // 😄拦截器已处理错误提示和日志，这里只需返回默认值
        return null
    }
}
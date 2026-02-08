// css-modules.d.ts 添加 CSS 模块类型声明
declare module "*.module.css" {
    const classes: { [key: string]: string };
    export default classes;
}

// 补充 SCSS Modules 类型声明
declare module "*.module.scss" {
    const classes: { [key: string]: string };
    export default classes;
}

// 补充 LESS Modules 类型声明
declare module "*.module.less" {
    const classes: { [key: string]: string };
    export default classes;
}
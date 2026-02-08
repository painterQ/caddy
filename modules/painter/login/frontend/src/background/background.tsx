import { useEffect, useRef, type PropsWithChildren } from "react";
import styles from "./background.module.css";
import init from "./scene";

interface BackgroundProp {

}

export default function Background({ children }: PropsWithChildren<BackgroundProp>) {
    const ref = useRef<HTMLDivElement>(null)
    useEffect(() => {
        if (ref.current) init(ref.current)
    }, [ref])
    return (
        <div ref={ref} style={{
            // backgroundImage: `url("${import.meta.env.BASE_URL}beautiful-scenery-svgrepo-com.svg")`,

        }}
            className={styles.background}
        >
            {children}
        </div>
    )
}

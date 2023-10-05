import './globals.css'
import { Inter } from 'next/font/google'

const inter = Inter({ subsets: ['latin'] })

export const metadata = {
  title: 'Muninn',
  description: 'Muninn MU Client',
}

export default function RootLayout({ children }) {
  return (
    <html className="h-full bg-white" lang="en">
      <body className={"h-full " + inter.className}>{children}</body>
    </html>
  )
}

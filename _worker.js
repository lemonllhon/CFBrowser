// Cloudflare Pages Advanced Mode entrypoint.
// The actual proxy logic stays in a separate file so it can also be deployed
// as an independent Worker in front of the Pages project.
import worker from './download-worker.js'

export default worker

# What is this

This repository houses code and content for my personal website, blog, portfolio and photo album, [rattz.xyz](https://rattz.xyz).

This is my corner of the internet, where I share personal and professional stuff. As such, it is always sprawling with new features and content (although I sometimes go months without revisiting the project).

# Handicaps

As to make development more fun, I've set a few restrictions for myself:

1. **No JavaScript:** There shall not be any JavaScript shipped to the client. This includes script tags, inline JavaScript, data/JS URLs, and all forms of client-side code execution.
1. **No Libraries:** I am only allowed to use Go's standard library (plus experimental) for implementing the server. This does not extend to other tools/languages used in the project, such as CSS or Terraform.
1. **No Slacking:** The pages should render as fast as possible. This is kind of a secondary restriction as I tend to implement features first and optimize later.

Overall, I'm also trying to make things more complicated\* while abiding to the aforementioned restrictions. A few examples of this are the weird caching policies I use for the content, the encoding of the image files (which I still want to make more _peculiar_), and the (in-development) "proprietary" templating language for blog posts.

\* I don't like introducing complexity for the sake of complexity. Most "complicated" things are just overkill features for the scale of the project or creative ways of doing regular tasks. I work alone so there is no harm in having some fun.

# To-dos

- Finish development of the Jambo templating language for the blog posts.
- Make it work with high-availability: the current plan is to use a discovery protocol between multiple instances and share local content.
- Write more blog posts.
- Reuse more code.
- Write tests.
- Improve gallery and codex logic. There are some rough edges still.
- After codex becomes the index: refactor to have a central struct that holds codex and gallery

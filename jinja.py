from flask import request, url_for


def use_jinja(app):

    @app.template_filter()
    def ym(num):
        text = str(num)
        return '{}-{}'.format(text[:4], text[4:])

    def url_for_other_page(page):
        args = request.view_args.copy()
        args['page'] = page
        return url_for(request.endpoint, **args)

    app.jinja_env.globals['url_for_other_page'] = url_for_other_page

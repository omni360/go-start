package user

import (
	"errors"
	"github.com/ungerik/go-start/view"
)

// The confirmation code will be passed in the GET parameter "code"
func EmailConfirmationView(profileURL view.URL) view.View {
	return view.DynamicView(
		func(response *view.Response) (view.View, error) {
			confirmationCode, ok := response.Request.Params["code"]
			if !ok {
				return view.DIV("error", view.HTML("Invalid email confirmation code!")), nil
			}

			doc, email, confirmed, err := ConfirmEmail(confirmationCode)
			if !confirmed {
				return view.DIV("error", view.HTML("Invalid email confirmation code!")), err
			}

			Login(response.Session, doc)

			return view.Views{
				view.DIV("success", view.Printf("Email address %s confirmed!", email)),
				&view.If{
					Condition: profileURL != nil,
					Content: view.P(
						view.HTML("Continue to your "),
						view.A(profileURL, "profile..."),
					),
				},
			}, nil
		},
	)
}

func NewLoginForm(buttonText, class, errorMessageClass, successMessageClass string, redirectURL view.URL) view.View {
	return view.DynamicView(
		func(response *view.Response) (v view.View, err error) {
			if from, ok := response.Request.Params["from"]; ok {
				redirectURL = view.StringURL(from)
			}
			model := &LoginFormModel{}
			if email, ok := response.Request.Params["email"]; ok {
				model.Email.Set(email)
			}
			form := &view.Form{
				Class:               class,
				ErrorMessageClass:   errorMessageClass,
				SuccessMessageClass: successMessageClass,
				SuccessMessage:      "Login successful",
				SubmitButtonText:    buttonText,
				FormID:              "gostart_user_login",
				GetModel:            view.FormModel(model),
				Redirect:            redirectURL,
				OnSubmit: func(form *view.Form, formModel interface{}, response *view.Response) (string, view.URL, error) {
					m := formModel.(*LoginFormModel)
					ok, err := LoginEmailPassword(response.Session, m.Email.Get(), m.Password.Get())
					if err != nil {
						if view.Config.Debug.Mode {
							return "", nil, err
						} else {
							return "", nil, errors.New("An internal error ocoured")
						}
					}
					if !ok {
						return "", nil, errors.New("Wrong email and password combination")
					}
					return "", nil, nil
				},
			}
			return form, nil
		},
	)
}

// If redirect is nil, the redirect will go to "/"
func LogoutView(redirect view.URL) view.View {
	return view.RenderView(
		func(response *view.Response) (err error) {
			Logout(response.Session)
			if redirect != nil {
				return view.Redirect(redirect.URL(response))
			}
			return view.Redirect("/")
		},
	)
}

// confirmationPage must have the confirmation code as first URL parameter
func NewSignupForm(buttonText, class, errorMessageClass, successMessageClass string, confirmationURL, redirectURL view.URL) *view.Form {
	return &view.Form{
		Class:               class,
		ErrorMessageClass:   errorMessageClass,
		SuccessMessageClass: successMessageClass,
		SuccessMessage:      Config.ConfirmationSent,
		SubmitButtonText:    buttonText,
		FormID:              "gostart_user_signup",
		GetModel: func(form *view.Form, response *view.Response) (interface{}, error) {
			return &EmailPasswordFormModel{}, nil
		},
		Redirect: redirectURL,
		OnSubmit: func(form *view.Form, formModel interface{}, response *view.Response) (string, view.URL, error) {
			m := formModel.(*EmailPasswordFormModel)
			email := m.Email.Get()
			password := m.Password1.Get()
			var user *User
			doc, found, err := FindByEmail(email)
			if err != nil {
				return "", nil, err
			}
			if found {
				user = From(doc)
				if user.EmailPasswordConfirmed() {
					return "", nil, errors.New("A user with that email and a password already exists")
				}
				user.Password.SetHashed(password)
			} else {
				user, _, err = New(email, password)
				if err != nil {
					return "", nil, err
				}
			}
			err = <-user.Email[0].SendConfirmationEmail(response, confirmationURL)
			if err != nil {
				return "", nil, err
			}
			return "", nil, user.Save()
		},
	}
}